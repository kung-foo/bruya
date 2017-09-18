package main

import (
	"net/http"
	"net/url"
	"os"

	_ "expvar"
	// TODO: make this optional?
	_ "net/http/pprof"

	docopt "github.com/docopt/docopt-go"
	"github.com/kung-foo/bruya"
	log "github.com/sirupsen/logrus"
)

// VERSION is set by the makefile
var VERSION = "0.0.0"

var usage = `
Usage:
    bruya [options] [-v ...] [--redis=<url>] [--nats=<url>] [--channel=<channel>...]
    bruya -h | --help | --version

Options:
    --redis=<url>           Redis URL [default: redis://localhost:6379]
    --nats=<url>            NATS Streaming URL [default: nats://localhost:4222]
    --channel=<channel>     Redis channel(s) to subscribe to [default: *]
    --nats-cluster-id=<id>  NATS cluster ID [default: test-cluster]
    --debug-http=<bind>     Enable pprof/expvar HTTP server.
                            Examples: "localhost:6060", ":6060"
    --debug-force-color     Force colorized logs.
    -h --help               Show this screen.
    --version               Show version.
    -v                      Enable verbose logging (-vv for very verbose)
`

var logger *log.Logger

func onErrorExit(err error) {
	if err != nil {
		logger.Fatalf("[bruya   ] %+v", err)
	}
}

func main() {
	mainEx(os.Args[1:])
}

func mainEx(argv []string) {
	logger = log.New()

	args, err := docopt.Parse(usage, argv, true, VERSION, true)
	onErrorExit(err)

	logger.Formatter = &log.TextFormatter{
		ForceColors: args["--debug-force-color"].(bool),
	}

	if args["-v"].(int) > 0 {
		logger.Level = log.DebugLevel
	}

	if args["-v"].(int) > 1 {
		// TODO: trace level
	}

	bruya.SetDefaultLogger(logger)

	logger.Debugf("[bruya   ] args: %+v", args)

	if args["--debug-http"] != nil {
		addr := args["--debug-http"].(string)
		logger.Infof("[bruya   ] starting debug server on http://%s/debug/pprof/", addr)

		go http.ListenAndServe(addr, nil)
	}

	r, err := url.Parse(args["--redis"].(string))
	onErrorExit(err)

	n, err := url.Parse(args["--nats"].(string))
	onErrorExit(err)

	bruya, err := bruya.New(&bruya.Options{
		ClusterID:         args["--nats-cluster-id"].(string),
		RedisURL:          r,
		NatsURL:           n,
		RedisChannelNames: args["--channel"].([]string),
	})

	onErrorExit(err)

	defer bruya.Stop()

	onErrorExit(bruya.Run())
}
