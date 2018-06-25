package bruya

import (
	"fmt"
	"net/url"

	"github.com/go-redis/redis"
	nats "github.com/nats-io/go-nats"
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
	Streaming         bool
	RedisURL          *url.URL
	ClusterID         string
	ClientID          string
	RedisChannelNames []string
	NameMangler       NameManglerFunc
}

type publisher interface {
	Publish(subj string, data []byte) error
	Close() error
}

type nconnwrapper struct {
	nconn *nats.Conn
}

func (n *nconnwrapper) Publish(subj string, data []byte) error {
	return n.nconn.Publish(subj, data)
}

func (n *nconnwrapper) Close() error {
	n.nconn.Close()
	return nil
}

type Bruya struct {
	publisher       publisher
	rconn           *redis.Client
	nameManglerFunc NameManglerFunc
	rchannels       []string
	done            chan struct{}
	//counter           *ratecounter.RateCounter
	//messagesPerSecond *expvar.Int
}

func New(options *Options) (*Bruya, error) {
	var err error
	var rconn *redis.Client

	defer func() {
		if err != nil {
			if rconn != nil {
				rconn.Close()
			}
		}
	}()

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

	rconn = redis.NewClient(roptions)

	_, err = rconn.Ping().Result()
	if err != nil {
		return nil, err
	}

	logger.Debugf("[bruya   ] connected to redis at: %s", options.RedisURL.String())

	// Note: we use a _new_ nuid because otherwise it shares the same prefix
	// as the one in the stan client. Not a big deal, but it looks visually
	// confusing. TODO: make a PR in stan to create their _own_ instance.
	if options.ClientID == "" {
		options.ClientID = fmt.Sprintf("bruya-%s", nuid.New().Next())
	}

	var p publisher

	if options.Streaming {
		if options.ClusterID == "" {
			options.ClusterID = "test-cluster"
		}
		noptions := stan.NatsURL(options.NatsURL.String())
		sconn, err := stan.Connect(options.ClusterID, options.ClientID, noptions)

		if err != nil {
			return nil, err
		}

		p = sconn

		logger.Debugf("[bruya   ] connected to nats streaming: %v", sconn.NatsConn().ConnectedUrl())
	} else {
		nconn, err := nats.Connect(options.NatsURL.String())

		if err != nil {
			return nil, err
		}

		p = &nconnwrapper{nconn}
		logger.Debugf("[bruya   ] connected to nats: %v", nconn.ConnectedUrl())
	}

	nmfn := options.NameMangler
	if nmfn == nil {
		nmfn = DefaultNameMangler
	}

	if len(options.RedisChannelNames) == 0 {
		options.RedisChannelNames = []string{"*"}
	}

	return &Bruya{
		rconn:           rconn,
		publisher:       p,
		done:            make(chan struct{}),
		nameManglerFunc: nmfn,
		rchannels:       options.RedisChannelNames,
		//counter:           ratecounter.NewRateCounter(time.Second),
		//messagesPerSecond: expvar.NewInt("messages_per_second"),
	}, nil
}

func (b *Bruya) Stop() error {
	close(b.done)
	b.publisher.Close()
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
			err = b.publisher.Publish(name, []byte(msg.Payload))
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
