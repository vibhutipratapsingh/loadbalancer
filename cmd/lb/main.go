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
	// read BACKENDS env or fallback
	backendsEnv := os.Getenv("BACKENDS")
	var initialBackends []string
	if backendsEnv == "" {
		initialBackends = []string{
			"http://localhost:8081",
			"http://localhost:8082",
			"http://localhost:8083",
		}
	} else {
		for _, b := range strings.Split(backendsEnv, ",") {
			if b = strings.TrimSpace(b); b != "" {
				initialBackends = append(initialBackends, b)
			}
		}
	}

	pool := lb.NewServerPool(initialBackends)

	// health checker (reuse your existing healthchecker implementation,
	// but ensure it calls pool.MarkHealth(addr, healthy))
	hc := lb.NewHealthChecker(pool)
	hc.Start(5 * time.Second)

	fmt.Println("🚀 Load Balancer running on :8080 (Least Connections + Health Checks)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Select using least-connections strategy
		backend, ok := pool.GetLeastConnBackend()
		if !ok {
			http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
			return
		}

		// increment active before forwarding
		pool.IncActive(backend)
		// ensure we decrement after request completes
		defer pool.DecActive(backend)

		// forward request synchronously (ForwardRequest returns after ServeHTTP completes)
		if err := proxy.ForwardRequest(backend, w, r); err != nil {
			// forwarding failure: decrement already scheduled via defer
			http.Error(w, "Failed to forward request", http.StatusBadGateway)
			return
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
