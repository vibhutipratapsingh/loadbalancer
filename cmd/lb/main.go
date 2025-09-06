package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"loadbalancer/lb"
	"loadbalancer/proxy"
)

func main() {
	backends := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	// Health checker
	hc := lb.NewHealthChecker(backends)
	hc.Start(5 * time.Second) // check every 5s

	// Round Robin (health-aware)
	rr := lb.NewRoundRobin(hc)

	fmt.Println("🚀 Load Balancer running on :8080 (Round Robin + Health Checks)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		backend, ok := rr.NextBackend()
		if !ok {
			http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
			return
		}
		if err := proxy.ForwardRequest(backend, w, r); err != nil {
			http.Error(w, "Failed to forward request", http.StatusInternalServerError)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
