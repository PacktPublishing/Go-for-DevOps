package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/pets/api"
)

var (
	// Pets is the set of pets the service will return
	Pets = []api.Pet{
		{
			Name:     "Thor",
			Type:     api.Dog,
			Birthday: time.Date(2021, 6, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			Name:     "Tron",
			Type:     api.Cat,
			Birthday: time.Date(2020, 7, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			Name:     "Goldie",
			Type:     api.Fish,
			Birthday: time.Date(2018, 2, 23, 0, 0, 0, 0, time.UTC),
		},
	}
)

func main() {
	m := http.NewServeMux()
	m.HandleFunc("/pets", petsHandler)
	wrappedHandler := otelhttp.NewHandler(m, "/pets")
	srv := &http.Server{
		Addr:    ":9000",
		Handler: wrappedHandler,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Print("Server started")

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	log.Print("Server shutting down...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
}

func petsHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("pets called")
	if err := json.NewEncoder(w).Encode(Pets); err != nil {
		w.WriteHeader(500)
		w.Write([]byte("failed to encode pets"))
		return
	}
}
