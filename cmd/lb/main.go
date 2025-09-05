package main

import (
	"fmt"
	"log"
	"net/http"

	"loadbalancer/lb"
	"loadbalancer/proxy"
)

func main() {
	// Hardcoded backends for now
	backends := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	// Initialize Round Robin load balancer
	rr := lb.NewRoundRobin(backends)

	fmt.Println("🚀 Load Balancer running on :8080 (Round Robin)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		backend := rr.NextBackend()
		if err := proxy.ForwardRequest(backend, w, r); err != nil {
			http.Error(w, "Failed to forward request", http.StatusInternalServerError)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
