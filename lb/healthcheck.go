package lb

import (
	"fmt"
	"net/http"
	"time"
)

// HealthChecker periodically checks backends from ServerPool
type HealthChecker struct {
	pool *ServerPool
}

// NewHealthChecker binds health checker to a server pool
func NewHealthChecker(pool *ServerPool) *HealthChecker {
	return &HealthChecker{pool: pool}
}

// Start runs health check goroutine
func (hc *HealthChecker) Start(interval time.Duration) {
	go func() {
		for {
			hc.checkAll()
			time.Sleep(interval)
		}
	}()
}

// checkAll pings all servers’ /health
func (hc *HealthChecker) checkAll() {
	for backend := range hc.pool.GetServers() {
		resp, err := http.Get(fmt.Sprintf("%s/health", backend))
		if err != nil || resp.StatusCode != http.StatusOK {
			hc.pool.MarkHealth(backend, false)
			fmt.Printf("❌ %s is DOWN\n", backend)
			continue
		}
		hc.pool.MarkHealth(backend, true)
		fmt.Printf("✅ %s is UP\n", backend)
	}
}
