# ChatOps Demonstration

This directory contains:

* A ChatOps service for Slack
* An traceable and metricized Petstore CRUD service
* Jaeger for Trace storage
* Prometheus for Metric storage

## Turnup the demo

In this directory, use the following command:

```bash
docker-compose up -d 
```

In a another terminal enter the following directory:
```
chapter/11/chatops/
```

Create a `.env` file with the following content:

```bash
AUTH_TOKEN=xoxb-[the rest of the token] 
APP_TOKEN=xapp-[the rest of the token] 
```

These were generated when we setup the Slack app. Please see chapter 11 for more details on generating these.

Run the bot with:

```bash
go run chatbotops.go 
```

In the background, there is a demo client that is adding pets to the petstore and doing searches for pets (some searches will cause errors). The service is set to Float sampling, so not every call will generate a trace. 

In another terminal, you can interact with the petstore by using the CLI application. This will let you add your own pets, delete pets, search for pets with a filter. That client can be found here: chapter/11/petstore/client/cli/petstore. You can find instructions on its use by running the following: 

```bash
go run petstore.go --help
```


## Turndown the demo

This consists of:

* Cancelling the running chatbot.go binary
* Running `docker-compose down` in this directory
