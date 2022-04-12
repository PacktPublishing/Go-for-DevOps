package main

import (
	"log"
	"net"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/server/grpc"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/server/http"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	lb, err := http.New()
	if err != nil {
		panic(err)
	}

	log.Println("load balancer started(8080)...")
	go func() {
		if err := lb.Serve(ln); err != nil {
			panic(err)
		}
	}()

	serv, err := grpc.New(":8081", lb)
	if err != nil {
		panic(err)
	}

	log.Println("grpc server started(8081)...")
	if err := serv.Start(); err != nil {
		panic(err)
	}
}
