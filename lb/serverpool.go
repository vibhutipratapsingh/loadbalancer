
import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Backend represents one backend server and its runtime metrics.
type Backend struct {
	Addr    string // e.g. "http://localhost:8081"
	Weight  int    // relative weight (>=1)
	active  int64  // active request count (atomic)
	Healthy bool   // health flag (protected by pool.mu)
}

// Active returns the current active connection count atomically.
func (b *Backend) Active() int64 {
	return atomic.LoadInt64(&b.active)
}

func (b *Backend) incActive() {
	atomic.AddInt64(&b.active, 1)
}

func (b *Backend) decActive() {
	n := atomic.AddInt64(&b.active, -1)
	if n < 0 {
		atomic.StoreInt64(&b.active, 0)
	}
}

// ServerPool manages backend servers and supports weighted selection.
type ServerPool struct {
	mu            sync.RWMutex
	backends      map[string]*Backend // key: backend Addr
	weightCounter uint64              // atomic counter used for weighted selection
}

// NewServerPoolFromMap initializes a pool from a map[address]weight.
func NewServerPoolFromMap(addrs map[string]int) *ServerPool {
	sp := &ServerPool{
		backends: make(map[string]*Backend, len(addrs)),
	}
	for addr, w := range addrs {
		if w <= 0 {
			w = 1
		}
		sp.backends[addr] = &Backend{Addr: addr, Weight: w, Healthy: true}
	}
	return sp
}

// NewServerPool (backwards compatible) - sets all weights to 1
func NewServerPool(addrs []string) *ServerPool {
	m := make(map[string]int, len(addrs))
	for _, a := range addrs {
		m[a] = 1
	}
	return NewServerPoolFromMap(m)
}

// AddServerWithWeight adds a backend with given weight.
func (sp *ServerPool) AddServerWithWeight(addr string, weight int) {
	if weight <= 0 {
		weight = 1
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.backends[addr] = &Backend{Addr: addr, Weight: weight, Healthy: true}
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
func (sp *ServerPool) GetServers() map[string]bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	copy := make(map[string]bool, len(sp.backends))
	for k, v := range sp.backends {
		copy[k] = v.Healthy
	}
	return copy
}

// HealthyBackends returns addresses of healthy backends.
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

// GetWeightedBackend selects a healthy backend according to weights.
// Returns addr and true, or "", false if none available.
func (sp *ServerPool) GetWeightedBackend() (string, bool) {
	// compute total weight
	total := sp.totalWeight()
	if total == 0 {
		return "", false
	}

	// increment atomic counter (lock-free)
	idx := atomic.AddUint64(&sp.weightCounter, 1)
	mod := int((idx - 1) % uint64(total)) // 0-based position in weight space

	// iterate healthy backends and find which weight bucket contains mod
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	cum := 0
	for _, b := range sp.backends {
		if !b.Healthy {
			continue
		}
		cum += b.Weight
		if mod < cum {
			return b.Addr, true
		}
	}
	// should never reach here if total>0, but return fallback
	return "", false
}

// IncActive increments the active counter for a backend address.
func (sp *ServerPool) IncActive(addr string) {
	sp.mu.RLock()
	b, ok := sp.backends[addr]
	sp.mu.RUnlock()
	if ok {
		b.incActive()
	} else {
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