# Agent - A System Agent Example

## Introduction

This is a an example of a System Agent from the "Go For DevOps" book by John Doak, David Justice and Sarah Murphy.

The agent defined here, which can be run by compiling and running the `agent/agent.go`program runs a system agent that exports system stats on port 8081 and runs a gRPC service that allows installing or removing software packages contained in a ".zip" file. 

These software packages are run in a container and setup on systemd within the user space that the agent runs on (it does not setup system level services).

## Running the agent

You can run the agent by compiling and deploying the "agent.go" file on a Linux box and then starting it. This agent is currently only Linux compatible.

## Running a client

There is a Cobra client located in `agent/client/cli` that you can compile and run from any device (saying that you compile it for the target platform). 

The Cobra client leverages a Go client at `agent/client` that can be used to programically access an endpoint (or set of endpoints to deploy on multiple machines at once).

We have included a sample application for you to install and run on the remote side, located in `agent/cli/sample/helloweb.zip`. 

## BEWARE

While this code runs and has been tested on an Ubuntu instance running on Azure, milleage may vary depending on systemd version and Linux distro.

In addition, this has not been setup in a production quality way.  For example, you will notice a lack of unit tests and integration tests. We have also not vetted all the containerized security parameters with Linux Kernel security experts. 

Finally, this misses features that should be in a production quality version, such as cgroup integration, passing of capabilities and forced network bindings.  

This is for an example only!