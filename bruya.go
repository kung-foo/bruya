package bruya

import (
	"fmt"
	"net/url"

	"github.com/go-redis/redis"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nuid"
	"github.com/pkg/errors"
)

type NameManglerFunc func(redisChannelName string) (natsSubjectName string)

func DefaultNameMangler(redisChannelName string) (natsSubjectName string) {
	return redisChannelName
}

type Options struct {
	NatsURL           *url.URL
	RedisURL          *url.URL
	ClusterID         string
	ClientID          string
	RedisChannelNames []string
	NameMangler       NameManglerFunc
}

type Bruya struct {
	sconn           stan.Conn
	rconn           *redis.Client
	nameManglerFunc NameManglerFunc
	rchannels       []string
	done            chan struct{}
	//counter           *ratecounter.RateCounter
	//messagesPerSecond *expvar.Int
}

func New(options *Options) (*Bruya, error) {
	if options.RedisURL.Scheme != "redis" {
		return nil, errors.Errorf("Invalid redis URL: %+v", options.RedisURL)
	}

	if options.NatsURL.Scheme != "nats" {
		return nil, errors.Errorf("Invalid nats URL: %+v", options.NatsURL)
	}

	roptions, err := redis.ParseURL(options.RedisURL.String())
	if err != nil {
		return nil, err
	}

	rconn := redis.NewClient(roptions)

	_, err = rconn.Ping().Result()
	if err != nil {
		return nil, err
	}

	logger.Debugf("[bruya   ] connected to redis at: %s", options.RedisURL.String())

	noptions := stan.NatsURL(options.NatsURL.String())

	// Note: we use a _new_ nuid because otherwise it shares the same prefix
	// as the one in the stan client. Not a big deal, but it looks visually
	// confusing. TODO: make a PR in stan to create their _own_ instance.
	if options.ClientID == "" {
		options.ClientID = fmt.Sprintf("bruya-%s", nuid.New().Next())
	}

	if options.ClusterID == "" {
		options.ClusterID = "test-cluster"
	}

	sconn, err := stan.Connect(options.ClusterID, options.ClientID, noptions)

	if err != nil {
		rconn.Close()
		return nil, err
	}

	logger.Debugf("[bruya   ] connected to nats streaming: %v", sconn.NatsConn().ConnectedUrl())

	nmfn := options.NameMangler
	if nmfn == nil {
		nmfn = DefaultNameMangler
	}

	if len(options.RedisChannelNames) == 0 {
		options.RedisChannelNames = []string{"*"}
	}

	return &Bruya{
		rconn:           rconn,
		sconn:           sconn,
		done:            make(chan struct{}),
		nameManglerFunc: nmfn,
		rchannels:       options.RedisChannelNames,
		//counter:           ratecounter.NewRateCounter(time.Second),
		//messagesPerSecond: expvar.NewInt("messages_per_second"),
	}, nil
}

func (b *Bruya) Stop() error {
	close(b.done)
	b.sconn.Close()
	b.rconn.Close()
	return nil
}

func (b *Bruya) Run() (err error) {
	ps := b.rconn.PSubscribe(b.rchannels...)
	psc := ps.Channel()

	defer func() {
		ps.PUnsubscribe(b.rchannels...)
		ps.Close()
	}()

	for {
		select {
		case msg := <-psc:
			name := b.nameManglerFunc(msg.Channel)
			err = b.sconn.Publish(name, []byte(msg.Payload))
			if err != nil {
				err = errors.Wrapf(err, "redis channel: \"%s\"", name)
				return
			}
			//b.counter.Incr(1)
			//b.messagesPerSecond.Set(b.counter.Rate())
		case <-b.done:
			return
		}
	}
}
