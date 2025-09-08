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
	// Example: read BACKENDS env in format "http://host1:8081=5,http://host2:8081=1"
	// Fallback example map below if BACKENDS not set.
	backendsEnv := os.Getenv("BACKENDS")
	var pool *lb.ServerPool

	if backendsEnv == "" {
		// default weights: backend1 weight 5, backend2 weight 2, backend3 weight 1
		pmap := map[string]int{
			"http://localhost:8081": 5,
			"http://localhost:8082": 2,
			"http://localhost:8083": 1,
		}
		pool = lb.NewServerPoolFromMap(pmap)
	} else {
		// parse env like "http://a:8081=5,http://b:8081=1"
		m := make(map[string]int)
		for _, part := range strings.Split(backendsEnv, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			// split on '='
			if strings.Contains(part, "=") {
				kv := strings.SplitN(part, "=", 2)
				addr := strings.TrimSpace(kv[0])
				var w int
				fmt.Sscanf(strings.TrimSpace(kv[1]), "%d", &w)
				if w <= 0 {
					w = 1
				}
				m[addr] = w
			} else {
				m[part] = 1
			}
		}
		pool = lb.NewServerPoolFromMap(m)
	}

	// Health checker should call pool.MarkHealth(addr, healthy)
	hc := lb.NewHealthChecker(pool) // make sure your HealthChecker expects ServerPool
	hc.Start(5 * time.Second)

	fmt.Println("🚀 Load Balancer running on :8080 (Weighted Round Robin + Health Checks)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		backend, ok := pool.GetWeightedBackend()
		if !ok {
			http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
			return
		}

		// mark active
		pool.IncActive(backend)
		defer pool.DecActive(backend)

		if err := proxy.ForwardRequest(backend, w, r); err != nil {
			http.Error(w, "Failed to forward request", http.StatusBadGateway)
			return
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
