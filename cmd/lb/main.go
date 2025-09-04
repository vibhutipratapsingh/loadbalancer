package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// backend servers (for now, we will just use one)
var backends = []string{
	"http://localhost:8081",
	"http://localhost:8082",
	"http://localhost:8083",
}

func main() {
	// Pick one backend (hardcoded for Day 2)
	target, err := url.Parse(backends[0])
	if err != nil {
		log.Fatal(err)
	}

	// Create a reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Handle all requests through proxy
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Forwarding request %s %s → %s", r.Method, r.URL.Path, target)
		proxy.ServeHTTP(w, r)
	})

	log.Println("Load Balancer started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
