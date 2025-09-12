package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"loadbalancer/lb"
	"loadbalancer/proxy"
)

// Default values
const (
	defaultListen = ":8080"
	defaultPort   = ":8081"
)

func main() {
	// Allow overriding listen address with a flag (useful for local testing)
	listen := flag.String("listen", defaultListen, "address to listen on (e.g. :8080)")
	flag.Parse()

	// Read BACKENDS env var. Format:
	//   http://host1:8081=5,http://host2:8081=1
	// or a simple comma list:
	//   http://host1:8081,http://host2:8081
	backendsEnv := os.Getenv("BACKENDS")

	var pool *lb.ServerPool
	if backendsEnv == "" {
		// fallback default map (weights set to 1)
		pmap := map[string]int{
			"http://localhost:8081": 1,
			"http://localhost:8082": 1,
			"http://localhost:8083": 1,
		}
		pool = lb.NewServerPoolFromMap(pmap)
	} else {
		// parse env string
		m := make(map[string]int)
		parts := strings.Split(backendsEnv, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if strings.Contains(p, "=") {
				kv := strings.SplitN(p, "=", 2)
				addr := strings.TrimSpace(kv[0])
				var w int
				fmt.Sscanf(strings.TrimSpace(kv[1]), "%d", &w)
				if w <= 0 {
					w = 1
				}
				m[addr] = w
			} else {
				m[p] = 1
			}
		}
		pool = lb.NewServerPoolFromMap(m)
	}

	// Strategy: weighted | least | roundrobin
	strategy := strings.ToLower(strings.TrimSpace(os.Getenv("STRATEGY")))
	if strategy == "" {
		strategy = "weighted" // default to weighted if weights exist
	}

	// Sticky: none | ip | cookie
	stickyMode := strings.ToLower(strings.TrimSpace(os.Getenv("STICKY")))
	if stickyMode == "" {
		stickyMode = "none"
	}

	// Optional server-side sticky map (for cookie mode)
	var stickyMap *lb.StickyMap
	if stickyMode == "cookie" {
		stickyMap = lb.NewStickyMap(30 * 24 * time.Hour)
	}

	// Start health checker that marks pool entries healthy/unhealthy.
	// Assumes lb.NewHealthChecker(pool) exists and its Start() will call pool.MarkHealth(...)
	hc := lb.NewHealthChecker(pool)
	hc.Start(5 * time.Second)

	log.Printf("🚀 Load Balancer starting on %s (strategy=%s sticky=%s)", *listen, strategy, stickyMode)

	// /stats endpoint: shows current servers and health (debugging)
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		servers := pool.GetServers() // map[string]bool
		// Build a richer view if ServerPool exposes backends info - fall back to this map
		out := make(map[string]any)
		out["servers"] = servers
		out["strategy"] = strategy
		out["sticky"] = stickyMode
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	// main handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1) If cookie sticky, try to honor existing cookie mapping
		if stickyMode == "cookie" {
			const cookieName = "LB-STICKY"
			if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
				// cookie stores clientID; stickyMap maps clientID -> backend
				if b, ok := stickyMap.Get(c.Value); ok {
					// check if assigned backend is healthy
					servers := pool.GetServers()
					if healthy, exists := servers[b]; exists && healthy {
						// use this backend
						pool.IncActive(b)
						defer pool.DecActive(b)
						if err := proxy.ForwardRequest(b, w, r); err != nil {
							http.Error(w, "Bad Gateway", http.StatusBadGateway)
						}
						return
					}
					// if assigned backend is unhealthy, fall through to select another backend
				}
			}
		}

		// 2) If IP-hash sticky mode
		if stickyMode == "ip" {
			clientIP := getClientIP(r)
			backends := pool.HealthyBackends()
			if len(backends) == 0 {
				http.Error(w, "No healthy backends", http.StatusServiceUnavailable)
				return
			}
			idx := lb.IpToIndex(clientIP, len(backends)) // expects exported IpToIndex
			if idx < 0 || idx >= len(backends) {
				// fallback to selection below
			} else {
				backend := backends[idx]
				pool.IncActive(backend)
				defer pool.DecActive(backend)
				if err := proxy.ForwardRequest(backend, w, r); err != nil {
					http.Error(w, "Bad Gateway", http.StatusBadGateway)
				}
				return
			}
		}

		// 3) Default selection based on strategy
		var backend string
		var ok bool

		switch strategy {
		case "weighted":
			backend, ok = pool.GetWeightedBackend()
		case "least":
			// prefer using pool.GetLeastConnBackend() if available
			backend, ok = pool.GetLeastConnBackend()
		case "roundrobin":
			// treat as weighted with all weights=1 (GetWeightedBackend will behave like RR)
			backend, ok = pool.GetWeightedBackend()
		default:
			// fallback
			backend, ok = pool.GetWeightedBackend()
		}

		if !ok || backend == "" {
			http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
			return
		}

		// If cookie sticky enabled, set mapping for the client
		if stickyMode == "cookie" {
			clientID := generateClientID(r)
			stickyMap.Set(clientID, backend)
			http.SetCookie(w, &http.Cookie{
				Name:     "LB-STICKY",
				Value:    clientID,
				Path:     "/",
				Expires:  time.Now().Add(30 * 24 * time.Hour),
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Secure:   false, // set true if using TLS
			})
		}

		// Mark active, forward, then decrement
		pool.IncActive(backend)
		defer pool.DecActive(backend)

		if err := proxy.ForwardRequest(backend, w, r); err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}
	})

	// Start HTTP server
	log.Fatal(http.ListenAndServe(*listen, nil))
}

// getClientIP extracts a client IP from request, preferring X-Forwarded-For.
func getClientIP(r *http.Request) string {
	// respect X-Forwarded-For (comma-separated list) if present
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		return strings.TrimSpace(parts[0])
	}
	// fallback to remote address
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// generateClientID produces a simple opaque client ID for cookie mapping.
// In production you might use a secure random UUID or sign/encrypt this value.
func generateClientID(r *http.Request) string {
	// Use timestamp + remote addr + UA – simple and reasonably unique for demo
	return fmt.Sprintf("%d-%s-%x", time.Now().UnixNano(), getClientIP(r), []byte(r.UserAgent())[:6])
}
