This provides an end to end test using docker to turn up Jaeger and an http server using Jaeger.

We kick off some traces using the client and then use our jaeger client to find the traces.

The test will kick off the docker-compose environment on its own.
