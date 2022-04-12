## Rollout Demo

### Setup

For this demo, you are going to need setup in which you have 1 system acting as a load balancer and other systems that act as backends.  We will be pushing out an update that:

- Removes a system form the load balancer
- Upgrades the binary on the remote system
- Tests the binary health
- Adds the system back to the load balancer

If no load balancer pool exists, it will add one. If one exists, the pool must be healthy and have the same backends as what we are upgrading according to our configuration file.

How you setup the test systems is an exercise for the reader. For testing we used Azure VM scale sets for the backends and a single VM for the load balancer. You could do this with local VMs on your workstation, physical machines or whatever means you want.

Here are the prerequisites:
- The system where the rollout is executed must have access to:
	- The system with the load balancer
	- All backends
	- Must have an SSH key file that can log into each system
- All systems must be able to communicate to each other on SSH
- Backends must be able to open port 8082
- LB must be able to open port 8080 and 8081
- Write a services.json file that represents our upgrade. Re-write the sample to your settings.

**Note**: Like most example code, this isn't production quality. This is simply meant as an example to teach lessons and show off capabilites you can use for your own nees.

### Running the LB

You will need to compile and push the rollout/lb binary to the system that will act as your load balancer. Running it is a simple as:

```bash
./lb
load balancer started(8080)...
grpc server started(8081)...
```

**Note**: The load balancer listens on all ports and has no security for gRPC. So don't expose this on an addressable IP.

### Prepare the rollout

You will need to compile the rollout binary and put it on whatever system is going to do the rollout.

In the same directory you will need to copy the webserver you want to deploy. You can use the one in `rollout/lb/sample` if you want. But whatever binary it is, it must:
- Answer with something on /
- Answer with "ok" on /healthz
- Run on port 8082

Finally, you need to have a `services.json` file that represents the rollout. Modify the sample one to your needs. If you have questions on the values, have a look at config.go where all the fields are detailed.

### Doing a rollout

With these files located in the same directory:
- rollout
- services.json
- webserver

You can simply do:
```bash
./rollout --keyFile=/home/[user]/.ssh/[key].pem services.json
```

This should start the process and do a rollout. The output will look like:

```
Setup LB with pool `/`
Starting Workflow
Running canary on: 10.0.0.5
Sleeping after canary for 1 minutes
Upgrading endpoint: 10.0.0.7
Upgrading endpoint: 10.0.0.6
Upgrading endpoint: 10.0.0.8
Upgrading endpoint: 10.0.0.9
Workflow Completed with no failures
```

### Common errors

```bash
Workflow Failed: esPreconditionFailure
Failed State  Error                                                                      
Precondition  checkLBState precondition fail: expected backends(5) != found backends(1) 
```
This usually occurs because you did a failed rollout. When a rollout happens, it expects either there to be an empty load balancer pool or all expected backends to be in and healthy.  You can just restart the "lb" binary and run again.


