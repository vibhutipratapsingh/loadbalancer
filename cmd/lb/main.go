package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"loadbalancer/lb"
	"loadbalancer/proxy"
)

func main() {
	// Read backends from env var BACKENDS (comma separated)
	// Example: "http://backend:8081,http://backend:8081"
	backendsEnv := os.Getenv("BACKENDS")
	var initialBackends []string
	if backendsEnv == "" {
		// fallback to localhost backends for local dev
		initialBackends = []string{
			"http://localhost:8081",
			"http://localhost:8082",
			"http://localhost:8083",
		}
	} else {
		for _, b := range strings.Split(backendsEnv, ",") {
			b = strings.TrimSpace(b)
			if b != "" {
				initialBackends = append(initialBackends, b)
			}
		}
	}

	// Server pool
	pool := lb.NewServerPool(initialBackends)

	// Health checker
	hc := lb.NewHealthChecker(pool)
	hc.Start(5 * time.Second)

	// Round Robin LB
	rr := lb.NewRoundRobin(pool)

	fmt.Println("🚀 Load Balancer running on :8080 (Round Robin + Server Pool)")

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
