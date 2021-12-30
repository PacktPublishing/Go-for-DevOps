# A Generic Workflow Service

![Diskerase runthrough](docs/images/client_runthrough.gif)
(Our diskerase client running a workflow on the server)

## Introduction

The example code layed out in this directory represents a generic workflow execution service. This service receives a protocol buffer to a gRPC service that represents the type of work we want to do (a WorkReq).

This is another example of separating the work to be done into two parts:

* Creation of the work in one program
* Execution of work in another

This allows centralization of all work done into a single system that can have security, emergency stop capabilities, policies, ... in a single location.  Clients that create work can have their own logic and tests. This benefits with:

* Central place to create reusable components
* Central policy enforcement
* One system authorized for changes instead of multiple
* Work logic is a separate system from work execution
* One place to stop bad things when they are occuring

The work is defined in `Block`s with one block executed at at time. Inside the `Block`s are `Job`s, which are the actions that are taken. Those will be executed concurrently within some rate limit you define for the `Block`.

Each `WorkReq` that is sent to the service is checked against a set of policies. If no policies are defined, the `WorkReq` is rejected. If the `WorkReq` violates a policy, it is rejected. Policies can be used to sanity check a `WorkReq`.

Once a `WorkReq` is received, a unique ID is generated and returned to the client. To execute that `WorkReq`, a second call to the server is made.

The server provides an RPC endoint to recover the status of the `WorkReq` for watching workflows execute.

`Job`s and `Policies` can be added to the system to expand its capabilities.

We include some sample data that is used to represents "sites", or places where machines are located.  We also include data that represents "machines" at those sites. These obey some naming conventions and I have included the generators I used to make this fake data.  These stand in for what would probably be a database or services that would hold authoritative information.

Finally I include a client that build `WorkReq` protocol buffers and call the service for a sample satellite disk erasure. You can use this to test that policies such as the token bucket work. You can alter these to try to defy the policies on the server.

## What this isn't

This is an example of a generic workflow system to demonstrate concepts from our chapter on "Designing for Chaos". 

This isn't a production quality service. If it has tests, they are not comprehensive. Unfortunately I have another full time job, so testing suffered for these book examples (something I would not do in my real job. Tests, tests and more tests!).

Other things that make it non-production quality:

* If we have a server restart, we cannot resume running workflows, we leave half eaten carcasses of workflows just laying around
* There is no security, so anyone could call this service. By default it starts on 127.0.0.1:8080 and doesn't have Jobs that do anything bad, but if you decide to change that, you need security
* Backend storage is local files in a temp directory
* Failures do not have some maximum count, they only stop work if a Job decideds they are fatal
* We don't write creations, start and end times
* There is no web interface
* Didn't provide a workflow killer except through emergency stop
* No pause capabilities
* No workflow cloning tools
* ...

The one example workflow I put in here is a diskerase for satellite datacenters. The `diskErase` `Job` isn't real, it just sleeps. The other jobs are simply looking at files representing information about fake sites and machines. These jobs could be made real, but for this demo I didn't want to actually mutate anything real.

You could turn this into a real system, but it would need some more bells and whistles.  This is a very lightweight version of a system I developed at Google. That service could handle service failures, restarts, horizontally scale and lots had lots of helpful packages... 

This is not that system.

## Structure overview

```
├── client
├── configs
├── data
│   ├── generators
│   │   └── mk
│   └── packages
│       └── sites
├── internal
│   ├── es
│   ├── policy
│   │   ├── config
│   │   └── register
│   │       ├── restrictjobtypes
│   │       ├── sameargs
│   │       └── startorend
│   ├── service
│   │   ├── executor
│   │   └── jobs
│   │       └── register
│   │           ├── diskerase
│   │           ├── sleep
│   │           ├── tokenbucket
│   │           └── validatedecom
│   └── token
├── proto
└── samples
    └── diskerase
        └── cmd
```

* `client/` contains a client library for talking to the service
* `configs/` contains server configuration files, like our policies and emergency stop
* `data/` contains fake data related to fake datacenters and machines
	* `generators/` has programs that generate our fake data
	* `packages/` has packages for reading our fake data
* `internal/` contains the server's internal packages
	* `es/` provides a package for reading emergency stop data
	* `policy/` defines our policy engine and registered policies
		* `config/` has a policy configuration file reader
		* `register/` has a policy register and sub-directories containing policies in the system
	* `service/` contains the service implementation
		* `executor/` holds the main execution engine for all workflows
			* `jobs` contains our job execution engine and all defined jobs in the system
				* `register/` has a job regiter and sub-directories containing jobs defined for the system
	* `token/` has a token bucket implemention
* `proto/` has the protocol buffer implementations used in the service, including how to define a workflow request
* `samples/` contains sample workflow creation programs that can submit to the workflow service
	* `diskerase/` contains a client for creating satellite disk erase workflows for the service to execute

## Finding Jobs that are available

The Jobs you can call are defined in: `internal/service/jobs/register/...`

Each file header in the directory will give informations such as:
```
Register name: "diskErase"
Args:
	"machine"(mandatory): The name of the machine, like "aa01" or "ab02"
	"site"(mandatory): The name of the site, like "aaa" or "aba"
Result:
	Erases a disk on a machine, except this is a demo, so it really just sleeps for 30 seconds.
```

This let's you know what arguements to use with a `Job` you define. So if this was a `Job` I wanted to call, I might do:

```go
job := &pb.Job{
	Name: "diskErase",
	Args: map[string]string{
		"machine": "aa01",
		"site": "aba02",
	}
}
```

You can see the `samples/diskerase` sample program to see a client program in action.

## Where to find policies

All policy implementations are define at: `internal/policy/register/...`

In each file you will see a call called: `policy.Register("startOrEnd", p)` where "startOrEnd" is the name of the policy. The `struct` called `Settings` will give all the settings for a policy to be applied to a workflow.

Policies to apply to a workflow are defined in: `configs/policies.json`

You must have a policy entry for every type of `WorkReq` you want to submit inside `configs/policies.json`. This is checked against `WorkReq.Name`.

## A satellite disk erasure client

You can find our example client that submits a datacenter satellite to have its disks erased at:
`samples/diskerase`

The following are instructions on running our `diskerase` client against the workflow system  (remember that this code doesn't actually erase any disks or do any real work, we are just faking it).

* Open a terminal
* Enter the `workflow/` directory
* Type: `go run workflow.go`

You should see some startup information like so:

```
Registered Job:  diskErase
Registered Job:  sleep
Registered Job:  tokenBucket
Registered Job:  validateDecom
Registered Policy:  restrictJobTypes
Registered Policy:  sameArgs
Registered Policy:  startOrEnd
Workflow Storage is at:  /var/folders/rd/hbhb8s197633_f8ncy6fmpqr0000gn/T/workflows
Server started
```

Now that we have the service running, let's start a satellite workflow:

* Open another terminal
* Enter the `workflow/samples/diskerase/` directory
* Type: `go run diskerase.go eraseSatellite aap`

This asks our `diskerase` client to create a `pb.WorkReq` representing a disk erasure for cluster "aap". Our client will do some pre-checks and then create the `pb.WorkReq`, submit it to the system and then ask the system to execute it. A file: "submit.log" will be created that holds any UUIDs for workflow you create.

It will then display a message like so:

![Diskerase status](docs/images/diskerase_status.png)

Once it has finished, you can access the full proto JSON output with:
`go run diskerase.go statusProto [workflow id]`

Or if you cancel out and want to resume watching, you can do:
`go run diskerase.go status [workflow id]`

## Some cool things to try

Now that you have seen the client and server, you can watch some of the concepts from the chaos chapter in action by trying to do things that you shouldn't.

Here are a few things to try out:

### Run a diskerase and then try to run another diskerase

This is the simplest thing to try, as it requires no changes to files. This should trigger the token bucket in the pre-conidtions block and fail. Only 1 of these can be triggered per hour.

### Run a diskerase on a non-satellite datacenter

This should fail because the tool only supports satellite datacenter types.

### Run a diskerase on a non-decom'd satellite

This should fail various pre-checks because it is not in a decom state.

### Make changes to es.json

Change `configs/es.json` so that the `diskErase` entry has `stop` instead of `go` while running a workflow. `es.go` checks that file every 10 seconds and the display refreshes every 10 seconds. You can watch the workflow stop.

You can try other things here like erasing the entry, which will have the same effect (or not having it in the right JSON format).

You can erase the entry and try submitting a new workflow, which will get denied.

### Make changes to the diskerase Jobs

You can change the Jobs that the `diskerase` client creates. You could add machines not in the same site, or remove precondition checks. These should violate policies and reject your jobs.

### Change the diskerase pb.WorkReq.Name

This should cause the service to not have a policy and automatically reject the workflow.
