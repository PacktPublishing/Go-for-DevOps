# Ops Service

The Ops service provides API access to various operational information that we want to allow various tool access to. While this could be built into the chatbot.go binary that implements the ChatOps interface, this abstraction allows us to have multiple programs that can access these API calls. In addition, if we want to migrate to another chat service instead of Slack (like Microsoft Teams), we can easily do so without impacting users during a migration.

## Basic Architecture

This is your standard gRPC service with a nice Go client ready made to access the service. We stick the important parts in `internal` to keep anyone from using the packages that are just for the service. `proto/` contains the protocol buffer messages we have for the client/server communication.

The file directory layout is as follows (with some highlighted files, but not all files):

```
├── client
│   └── client.go
├── internal
│   ├── jaeger
│   │   └── client
│   │       ├── client.go
│   │       └── test
│   ├── prom
│   │   └── prom.go
│   └── server
│       └── server.go
├── ops.go
└── proto
    ├── jaeger
    │   ├── model
    ├── ops.pb.go
    ├── ops.proto
    └── ops_grpc.pb.go
```

* `ops.go` is the main file for the Ops service
* `client/ provides` a client library for accessing our Ops service using Go
* `internal/jaeger` provides a client wrapper for accessing Jaeger and some end to end testing
* `internal/prom` provides a client wrapper for accessing prometheus
* `proto/` contains protocol buffer messages and services for accessing the Ops service via gRPC
	* `proto/jaeger` provides various protocol buffers required to access Jaeger
