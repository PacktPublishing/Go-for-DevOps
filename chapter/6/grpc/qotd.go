package main

import (
	"flag"
	"log"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/6/grpc/server"
)

var addr = flag.String("addr", "127.0.0.1:80", "The address to run on.")

func main() {
	flag.Parse()

	s, err := server.New(*addr)
	if err != nil {
		panic(err)
	}

	done := make(chan error, 1)

	log.Println("Starting server at: ", *addr)
	go func() {
		defer close(done)
		done <- s.Start()
	}()

	err = <-done
	log.Println("Server exited with error: ", err)
}
