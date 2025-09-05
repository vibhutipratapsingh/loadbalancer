package lb

import (
	"sync/atomic"
)

// RoundRobin struct manages a list of backends and an atomic counter
type RoundRobin struct {
	backends []string
	counter  uint64
}

// NewRoundRobin creates a new RoundRobin instance
func NewRoundRobin(backends []string) *RoundRobin {
	return &RoundRobin{
		backends: backends,
	}
}

// NextBackend returns the next backend in round robin order
func (rr *RoundRobin) NextBackend() string {
	idx := atomic.AddUint64(&rr.counter, 1)
	return rr.backends[(idx-1)%uint64(len(rr.backends))]
}
