package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// take port from CLI flag
	port := flag.String("port", "8081", "server port")
	flag.Parse()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		msg := fmt.Sprintf("Hello from backend server on port %s\n", *port)
		w.Write([]byte(msg))
	})

	addr := fmt.Sprintf(":%s", *port)
	log.Printf("Backend server started on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
