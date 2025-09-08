package lb

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Backend represents one backend server and its runtime metrics.
type Backend struct {
	Addr    string // e.g. "http://localhost:8081"
	active  int64  // active request count (access via atomic)
	Healthy bool   // health flag (protected by pool.mu)
}

// Active returns the current active connection count atomically.
func (b *Backend) Active() int64 {
	return atomic.LoadInt64(&b.active)
}

// incActive increments active connections by 1.
func (b *Backend) incActive() {
	atomic.AddInt64(&b.active, 1)
}

// decActive decrements active connections by 1 (not below 0).
func (b *Backend) decActive() {
	// use Add with -1 to decrement
	n := atomic.AddInt64(&b.active, -1)
	if n < 0 {
		// safety: reset to 0 if it ever goes negative
		atomic.StoreInt64(&b.active, 0)
	}
}

// ServerPool manages a set of Backend entries with concurrency safety.
type ServerPool struct {
	mu       sync.RWMutex
	backends map[string]*Backend // key: backend Addr
}

// NewServerPool initializes a server pool from given backend addresses.
func NewServerPool(addrs []string) *ServerPool {
	sp := &ServerPool{
		backends: make(map[string]*Backend, len(addrs)),
	}
	for _, a := range addrs {
		sp.backends[a] = &Backend{Addr: a, Healthy: true}
	}
	return sp
}

// AddServer adds a new backend to the pool and marks it healthy.
func (sp *ServerPool) AddServer(addr string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.backends[addr] = &Backend{Addr: addr, Healthy: true}
}

// RemoveServer removes a backend from the pool.
func (sp *ServerPool) RemoveServer(addr string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	delete(sp.backends, addr)
}

// MarkHealth marks a backend as healthy/unhealthy.
func (sp *ServerPool) MarkHealth(addr string, healthy bool) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if b, ok := sp.backends[addr]; ok {
		b.Healthy = healthy
	}
}

// GetServers returns a copy of all backends with health state.
// Useful for health checker iteration.
func (sp *ServerPool) GetServers() map[string]bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	copy := make(map[string]bool, len(sp.backends))
	for k, v := range sp.backends {
		copy[k] = v.Healthy
	}
	return copy
}

// HealthyBackends returns a slice of addresses for currently healthy backends.
func (sp *ServerPool) HealthyBackends() []string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	out := make([]string, 0, len(sp.backends))
	for addr, b := range sp.backends {
		if b.Healthy {
			out = append(out, addr)
		}
	}
	return out
}

// ------------------ Least Connections Strategy API ------------------

// GetLeastConnBackend returns the address of the healthy backend with the lowest
// active connection count. Returns "" and false if no healthy backend exists.
//
// This is the core of the "Least Connections" load-balancing strategy.
func (sp *ServerPool) GetLeastConnBackend() (string, bool) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	var chosen *Backend
	for _, b := range sp.backends {
		if !b.Healthy {
			continue
		}
		if chosen == nil {
			chosen = b
			continue
		}
		// compare active counts; choose the one with fewer active connections
		if b.Active() < chosen.Active() {
			chosen = b
		}
	}

	if chosen == nil {
		return "", false
	}
	return chosen.Addr, true
}

// IncActive increments the active counter for a backend address.
func (sp *ServerPool) IncActive(addr string) {
	sp.mu.RLock()
	b, ok := sp.backends[addr]
	sp.mu.RUnlock()
	if ok {
		b.incActive()
	} else {
		// optional: log unknown addr
		fmt.Printf("IncActive: unknown backend %s\n", addr)
	}
}

// DecActive decrements the active counter for a backend address.
func (sp *ServerPool) DecActive(addr string) {
	sp.mu.RLock()
	b, ok := sp.backends[addr]
	sp.mu.RUnlock()
	if ok {
		b.decActive()
	} else {
		fmt.Printf("DecActive: unknown backend %s\n", addr)
	}
}
