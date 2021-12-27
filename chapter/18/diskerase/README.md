# A Generic Workflow Service

## Introduction

The example code layed out in this directory represents a generic workflow execution service. This service receives a protocol buffer to a gRPC service that represents the type of work we want to do (a WorkReq).

This is another example of separating the work to be done into two parts:
* Creation of the work in one program
* Execution of work

This allows centralization of all work done into a single system that can have security, emergency systems, policies, ... in a single location.  Things that create work can have their own logic and tests. This benefits with:
* Central place to create reusable components
* Central policy enforcement
* One system authorized for changes instead of multiple
* Work logic is separate system from work execution
* One place to stop bad things when the are occuring

The work is defined in `Block`s with one block executed at at time. Inside the `Block`s are `Job`s, which are the actions that are taken. Those will be executed concurrently within some rate limit you define for the `Block`.

Each `WorkReq` that is sent to the service is checked against a set of policies. If no policies are defined, the `WorkReq` is rejected. If the `WorkReq` violates a policy, it is rejected. Policies can be used to sanity check a `WorkReq`.

Once a `WorkReq` is received, a unique ID is generated and returned to the client. To execute that `WorkReq`, a second call to the server is made to tell it to execute.

The server provides an RPC endoint to recover the status of the `WorkReq`.

`Job`s and `Policies` can be added to the system to expand its capabilities.

We include some sample data that is used to represents "sites", or places where machines are located.  We also include data that represents "machines" at those sites. These obey some naming conventions and I have included the generators I used to make this fake data.  These stand in for what would probably be database or services that would hold this data.

Finally I include some clients that build `WorkReq` protocol buffers and call the service. You can use these to tests that policies such as the token bucket work. You can alter these to try to defy the policies on the server.

## What this isn't

This is an example of a generic workflow system to demonstrate a bunch of concepts. This isn't a production quality service. If it has tests, they are not comprehensive. Unfortunately I have another full time job, so testing suffered for these book examples (something I would not do in my real job. Tests, tests and more tests!).

Other things that make it non-production quality:
* If we have a server restart, we cannot resume running workflows. 
* There is no security, so anyone could call this service.  
* Backend storage is local files. 
* Failures do not have some maximum count, they only stop work if a Job decideds they are fatal.

The one example workflow I put in here is a diskerase for satellite datacenters. The `diskErase` `Job` isn't real, it just sleeps.  

You could turn this into a real system, but it would need a lot more bells and whistles.  This is a very lightweight version of a system I developed at Google. That service could handle service failures, restarts, horizontally scaled, ... 

This is not that system.

## Service proto definitions

You can find these at: `proto/diskerase.proto`

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

This let's you know what to call in with a `Job` you define. So if this was a `Job` I wanted to call, I might do:

```go
job := &pb.Job{
	Name: "diskErase",
	Args: map[string]string{
		"machine": "aa01",
		"site": "aba02",
	}
}
```

## Policies

`Policies` are defined in: `configs/policies.json`

You must have a policy for every type of WorkReq you want to submit. This is checked against `WorkReq.Name`.

All policy types are define at: `internal/policy/register/...`

In each file you will see a call called: `policy.Register("startOrEnd", p)` where "startOrEnd" is the name of the policy. The `struct` called `Settings` will give all the settings for a policy.

You can call any of these policies from `policies.json` for a given `WorkReq.Name`.

