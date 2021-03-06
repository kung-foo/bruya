# bruya

Forward Redis pub/sub messages to a NATS or NATS Streaming endpoint.

```
Usage:
    bruya [options] [-v ...] [--redis=<url>] --nats=<url> [--channel=<channel>...]
    bruya [options] [-v ...] [--redis=<url>] --stan=<url> [--channel=<channel>...]
    bruya -h | --help | --version

Options:
    --redis=<url>           Redis URL [default: redis://localhost:6379]
    --nats=<url>            NATS URL
    --stan=<url>            NATS Streaming URL
    --channel=<channel>     Redis channel(s) to subscribe to [default: *]
    --nats-cluster-id=<id>  NATS cluster ID [default: test-cluster]
    --debug-http=<bind>     Enable pprof/expvar HTTP server.
                            Examples: "localhost:6060", ":6060"
    --debug-force-color     Force colorized logs.
    -h --help               Show this screen.
    --version               Show version.
    -v                      Enable verbose logging (-vv for very verbose)
```
