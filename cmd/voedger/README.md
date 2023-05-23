Provide CLI logic for CE

# Basic usage

- help: `go run main.go`
- run server:
  - windows: `go run main.go --ihttp.Port 8888 server`
  - linux: `/usr/local/.go/bin/go run main.go --ihttp.Port 8888 server`
- work with server
  - try static resources - open http://localhost:8888/static/sys/monitor/site/hello/
  - monitor - open http://localhost:8888/static/sys/monitor/site/main/