# go-check-prometheus

A nagios check for prometheus inspired from 
[go-check-graphite](https://github.com/SegFaultAX/go-check-graphite) but for prometheus.

## Installation

`go install github.com/shashankm/go-check-prometheus`

## Usage

```shell
go-check-prometheus -H prometheus.example.com:9090 -q 'my.query' -w 10 -c 100
```

## Documentation

```
usage: go-check-prometheus [options]

Options:
  -c, --critical string   critical range
  -h, --help              show help
  -H, --host string       prometheus host
  -n, --name string       Short, descriptive name for metric (default "metric")
  -q, --query string      prometheus query
  -t, --timeout int       Execution timeout (default 10)
  -w, --warning string    warning range
```
