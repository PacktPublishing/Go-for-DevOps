package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var port = flag.Int("port", 8080, "The port to run on")

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello web")
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
