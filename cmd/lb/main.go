package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

// ==============================
// CONFIGURATION
// ==============================

// List of backend servers your LB will distribute traffic to.
// Later we’ll make this dynamic (config file / hot reload),
// but for now it's hardcoded for simplicity.
var backends = []string{
	"http://localhost:8081",
	"http://localhost:8082",
	"http://localhost:8083",
}

// Atomic counter keeps track of request number.
// We use uint64 to avoid overflow issues.
var counter uint64

// ==============================
// ROUND ROBIN STRATEGY
// ==============================

// getNextBackend returns the next backend server using round robin.
//
// Example with 3 servers:
// Request 1 → backend[0]
// Request 2 → backend[1]
// Request 3 → backend[2]
// Request 4 → backend[0] (cycle repeats)
//
// We use atomic.AddUint64() so multiple goroutines
// can increment safely without race conditions.
func getNextBackend() string {
	idx := atomic.AddUint64(&counter, 1)           // increment counter atomically
	return backends[(idx-1)%uint64(len(backends))] // modulo ensures cycling
}

// ==============================
// HTTP HANDLER
// ==============================

// handler accepts the incoming client request and forwards
// it to the selected backend server using httputil.ReverseProxy.
func handler(w http.ResponseWriter, r *http.Request) {
	// 1. Pick a backend using round robin
	backend := getNextBackend()

	// 2. Parse backend URL (required by ReverseProxy)
	target, err := url.Parse(backend)
	if err != nil {
		http.Error(w, "Bad backend URL", http.StatusInternalServerError)
		return
	}

	// 3. Create a reverse proxy to forward the request
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 4. Log which backend this request was sent to
	fmt.Printf("Forwarding request %s → %s\n", r.URL.Path, backend)

	// 5. Forward request + response back to client
	proxy.ServeHTTP(w, r)
}

// ==============================
// MAIN FUNCTION
// ==============================

func main() {
	fmt.Println("🚀 Load Balancer running on :8080 (Round Robin)")

	// Register handler for all routes
	http.HandleFunc("/", handler)

	// Start LB server on port 8080
	// If it crashes, log.Fatal will show the error
	log.Fatal(http.ListenAndServe(":8080", nil))
}
