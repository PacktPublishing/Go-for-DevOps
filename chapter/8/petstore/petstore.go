package main

import (
	"flag"
	"log"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/server"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/server/storage/mem"
)

var addr = flag.String("addr", "127.0.0.1:6742", "The address to run on.")

func main() {
	flag.Parse()

	store := mem.New()

	s, err := server.New(*addr, store)
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
