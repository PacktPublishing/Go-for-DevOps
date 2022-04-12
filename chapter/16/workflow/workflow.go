package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/data/packages/sites"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/policy/config"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/service"
	"google.golang.org/grpc"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"

	// These register all our job types, as each has an init() that registers the Job with
	// the service. This is called a side effects import, because we don't actually use it.
	// The _ before the package indicates it will not be used directly.
	_ "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/service/jobs/register/diskerase"
	_ "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/service/jobs/register/sleep"
	_ "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/service/jobs/register/tokenbucket"
	_ "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/service/jobs/register/validatedecom"

	// These register all our policies, exactly like our Jobs work.
	_ "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/policy/register/restrictjobtypes"
	_ "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/policy/register/sameargs"
	_ "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/policy/register/startorend"
)

var (
	addr = flag.String("addr", "127.0.0.1:8080", "The address to run the server on")
)

// dirMode is simply the mode we create our directories with.
const dirMode = os.ModeDir | 0700

func main() {
	flag.Parse()

	// Read our policy config.
	config.Init()
	sites.Init("data")

	// This makes sure we have a place to store workflows.
	p := filepath.Join(os.TempDir(), "workflows")

	stat, err := os.Stat(p)
	if err == nil {
		if !stat.IsDir() {
			panic(p + " is not a direcotry")
		}
		if stat.Mode() != os.FileMode(dirMode) {
			panic(fmt.Sprintf("%s is mode %v, not %v", p, stat.Mode(), dirMode))
		}
	} else {
		if err := os.Mkdir(p, dirMode); err != nil {
			panic(fmt.Sprintf("could not create directory(%s): %s", p, err))
		}
	}
	log.Println("Workflow Storage is at: ", p)

	// Create our implementation of the gRPC service.
	serv, err := service.New(p)
	if err != nil {
		panic(err)
	}

	// Create a new gRPC service and register our implementation.
	g := grpc.NewServer()
	pb.RegisterWorkflowServer(g, serv)

	// Grab our address on the network and begin listening.
	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		panic(err)
	}

	// Tell gRPC to use our listener for new connections.
	log.Println("Server started on: ", *addr)
	g.Serve(lis)
}
