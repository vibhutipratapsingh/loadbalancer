package lb

import (
	"sync/atomic"
)

// RoundRobin uses an atomic counter + health checker
type RoundRobin struct {
	hc      *HealthChecker
	counter uint64
}

// NewRoundRobin creates RoundRobin with health checking
func NewRoundRobin(hc *HealthChecker) *RoundRobin {
	return &RoundRobin{hc: hc}
}

// NextBackend picks the next healthy backend.
// Returns false if none available.
func (rr *RoundRobin) NextBackend() (string, bool) {
	backends := rr.hc.HealthyBackends()
	if len(backends) == 0 {
		return "", false // all servers down
	}
	idx := atomic.AddUint64(&rr.counter, 1)
	return backends[(idx-1)%uint64(len(backends))], true
}
