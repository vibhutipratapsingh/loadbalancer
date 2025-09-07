package lb

import (
	"sync"
)

// ServerPool manages backend servers dynamically (add/remove/health check)
type ServerPool struct {
	servers map[string]bool // backend → healthy/unhealthy
	mu      sync.RWMutex
}

// NewServerPool creates a server pool with initial backends
func NewServerPool(backends []string) *ServerPool {
	sp := &ServerPool{servers: make(map[string]bool)}
	for _, b := range backends {
		sp.servers[b] = true // assume all healthy at start
	}
	return sp
}

// AddServer adds a backend to the pool
func (sp *ServerPool) AddServer(backend string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.servers[backend] = true
}

// RemoveServer removes a backend from the pool
func (sp *ServerPool) RemoveServer(backend string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	delete(sp.servers, backend)
}

// MarkHealth updates server health status
func (sp *ServerPool) MarkHealth(backend string, healthy bool) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.servers[backend] = healthy
}

// GetServers returns all servers with their health status
func (sp *ServerPool) GetServers() map[string]bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	// return a copy to avoid race conditions
	copy := make(map[string]bool)
	for k, v := range sp.servers {
		copy[k] = v
	}
	return copy
}

// HealthyBackends returns a slice of only healthy servers
func (sp *ServerPool) HealthyBackends() []string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	var list []string
	for backend, ok := range sp.servers {
		if ok {
			list = append(list, backend)
		}
	}
	return list
}
