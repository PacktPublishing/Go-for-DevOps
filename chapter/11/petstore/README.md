# Petstore Application

## Introduction

In chapter 8, you have seen an example application that did metrics and tracing using Open Telemetry (OTEL). This application was designed to be simple and easy to understand.

This application is how a traditional application would be instrumented. This application is based on gRPC platform that we have used before. We track more useful metrics to allow us to detect problems with our application.

We have removed most standard logging mechansisms (and no use of zero logger), instead we provide our own log package that writes our log message into the current span. That allows us to narrow our logging noise and avoid all the correlation required if you use standard logging. We only log startup information that is relevant before out service is ready (what ports we ran on, we can't start OTEL, startup args, ...).

We have spun off our own error package that logs our errors to the span and generates the error type.  This means we don't have to figure out when to log errors, they are always logged to spans. We only use standard errors when the error is not being generated in a span path.

We have moved our metrics and tracing constructors to their own packages and out of the main package. This lets us offer multiple places to put our traces or metrics. In the case of tracing, we offer a stderr tracing provider or provider that traces to a file.

Finally, we provide our own tracing sampler which wraps one of the standard samplers. This allows us to trace whenever an RPC has the "trace" key in the gRPC request metadata or we receive one with a TraceID set. Otherwise we can do sampling at some rate, for ever RPC or not trace at all. Our sampler can be dialed up or down and this can be down with a management RPC we provide to allow changing our sampling.

## Running

- `docker-compose up -d` (if you remove -d, you will see all the logs from the docker jobs in stdout, ^c to make it stop)
- Once started the client application will periodically add pets to the server until it runs out of names to add
- Metrics will be collected for various things
- Traces happen at 10% sampling rate
- You can use the cli/petstore application to query the service yourself

A sample query for all felines that have birthday's after Jan 1, 2004:

`machine:.../client/petstore$ go run petstore.go search types="PTFeline" birthdayStart='{"month":1, "day":1, "year":2004}'

If you want to force the query to do a trace, you can add `--trace` after `petstore.go`.

Prometheus has metrics at: http://localhost:9090
Traces are in Jaegar at: http://localhost:16686


If you see something like:
```bash
docker-compose up -d
Traceback (most recent call last):
  File "urllib3/connectionpool.py", line 670, in urlopen
  File "urllib3/connectionpool.py", line 392, in _make_request
```
This indicates that you aren't running docker. Make sure you have docker installed and it is running. 

## Teardown

Simple run; `docker-compose down`

## Structure

Here is a breakdown of how the petstore application is made. This is not required for the chapter's use, but looking at the code will give you a deeper insight into how you might want to structure a real application.

```
├── client
│   ├── cli
│   │   └── petstore
│   │       └── petstore.go
│   └── client.go
├── internal
│   └── server
│       ├── errors
│       ├── log
│       ├── server.go
│       ├── storage
│       │   ├── mem
│       │   └── storage.go
│       └── telemetry
│           ├── metrics
│           └── tracing
│               ├── sampler
│               └── tracing.go
├── petstore.go
└── proto
```

* client/ Has an RPC client for the service
* client/cli/petstore Is a CLI client to send RPCs to the petstore
* petstore/ Is the main package
* internal/server Is the gRPC service implementation
* internal/server/errors The app's error package, works similar to the "errors" package from stdlib
* internal/server/log The app's logging pacakge, similar to "log" from the stdlib
* storage/ Defines the storage abstraction for the service
* storage/mem Defines an in-memory storage implementation of storage.Data
* telemetry/metrics Defines all the OpenTelemetry(OTEL) metrics for the application
* telemetry/tracing Defines the Opentelemetry(OTEL) tracing for the application
* proto/ Contains our protocol buffer definitions and Go packages

Of note:
	* The log package outputs log messages to a tracing Span, not to a log file
	* If you do need to output some startup info, log.Logger is the standard log.Logger type, defaults to the default Logger
	* The errors package is a drop in replacement for "log", except Errorf() and New() take a Context
	* errors will write the error out to the current Span


