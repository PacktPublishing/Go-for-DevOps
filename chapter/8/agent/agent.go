/*
The agent application is run on a remote system under a user you will connect to via RPC
and issue commands.

The applications installed will containerized within that user's space. Even though they
are in a container, it is best to use a non-priviledged user.

This also exports sytsem stats on port :8081. There is no security on this web export,
just an FYI if this system is exposed directly to the internet.
*/
package main

import (
	"log"
	"net/http"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/agent/internal/service"
)

func main() {
	agent, err := service.New()
	if err != nil {
		panic(err)
	}

	go func() {
		err := http.ListenAndServe(":8081", nil)
		panic(err)
	}()

	log.Println("Service starting...")
	if err := agent.Start(); err != nil {
		panic(err)
	}
}
