## Load Balancer Example

We have added a simple layer 7 Load Balancer here to use in the rollout example from the book. This code is not covered in that chapter, instead it simply covers that you are interacting with a load balancer to add and remove web servers.

To make the full example work, we have built a simple load balancer using Go with client libraries and sample applications.

There is also a CLI application for interacting with the load balancer outside the example application.

### Using the Load Balancer

The load balancer is setup to run on port 8080. Your OS will need to allow this port and nothing else can be running on it.

To run the load balancer, in this directory simply run:
```bash
go run lb.go
```

### Interacting with the CLI

We included a CLI app, which is not required for the example. It allowed us to do basic testing with the load balancer.

The CLI app will allow you to:
- Add a pool, which is based on the URL pattern to match against
- Remove a pool by pattern
- Add a backend to a pool
- Remove a backend from a pool
- Get a pools health

Pattern matching is based on the `http` package pattern matching. See the GoDoc for more information.

Adding a pool with two backends looks like:
```go
$ go run cli.go --lb=127.0.0.1:8081 --pattern=/ addPool
$ go run cli.go --lb=127.0.0.1:8081 --pattern=/ --ip=127.0.0.1 --port=8082 --url_path=/ addBackend
$ go run cli.go --lb=127.0.0.1:8081 --pattern=/ --ip=127.0.0.1 --port=8083 --url_path=/ addBackend
```
This first contacts our load balancer (127.0.0.1:8081) and adds a pool that matches pattern /.

Then it adds two backends running on local ports 8082 and 8083 with a URL path of /.

Note that the CLI sets up a health check that queries the backend's `/healthz` page looking for `ok` in the body of the response. If it doesn't respond, you can't add that backend.  This is also checked at intervals and it will remove unhealthy nodes until they pass a health check.

There is an example web server you can run in the `sample/web` directory to provide the load balancer with valid backends. Simply go into that directory and run:
```bash
$ go run main.go --port=8082
```
This would run a webserver on port 8082.  You can do this multiple times on different ports. In the example above we ran them on 8082 and 8083.

You can see the health of your pool with:
```bash
go run cli.go --lb=127.0.0.1:8081 --pattern=/ poolHealth
Pool  Status   
/     PS_FULL  
Backend         Status      
127.0.0.1:8082  BS_HEALTHY  
127.0.0.1:8083  BS_HEALTHY  
```

### NOTES

- This is not a production level load balancer. It lacks a lot of bells and whistles, monitoring, metrics and most importantly tests.
- There is no security on the gRPC service.