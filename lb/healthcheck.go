package lb

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HealthChecker keeps track of which backends are healthy.
type HealthChecker struct {
	servers map[string]bool // backend URL → healthy/unhealthy
	mu      sync.RWMutex
}

// NewHealthChecker creates a new health checker with initial backends marked as healthy.
func NewHealthChecker(backends []string) *HealthChecker {
	hc := &HealthChecker{servers: make(map[string]bool)}
	for _, b := range backends {
		hc.servers[b] = true // assume healthy at start
	}
	return hc
}

// IsHealthy checks if a backend is currently marked healthy.
func (hc *HealthChecker) IsHealthy(backend string) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.servers[backend]
}

// HealthyBackends returns only the backends that are currently healthy.
func (hc *HealthChecker) HealthyBackends() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	var list []string
	for backend, ok := range hc.servers {
		if ok {
			list = append(list, backend)
		}
	}
	return list
}

// Start launches a goroutine that pings /health every interval.
func (hc *HealthChecker) Start(interval time.Duration) {
	go func() {
		for {
			hc.checkAll()
			time.Sleep(interval)
		}
	}()
}

// checkAll makes a GET request to each backend’s /health endpoint.
func (hc *HealthChecker) checkAll() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	for backend := range hc.servers {
		resp, err := http.Get(fmt.Sprintf("%s/health", backend))
		if err != nil || resp.StatusCode != http.StatusOK {
			hc.servers[backend] = false
			fmt.Printf("❌ %s is DOWN\n", backend)
			continue
		}
		hc.servers[backend] = true
		fmt.Printf("✅ %s is UP\n", backend)
	}
}
