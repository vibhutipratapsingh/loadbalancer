package lb

import (
	"sync/atomic"
)

// RoundRobin with ServerPool
type RoundRobin struct {
	pool    *ServerPool
	counter uint64
}

func NewRoundRobin(pool *ServerPool) *RoundRobin {
	return &RoundRobin{pool: pool}
}

func (rr *RoundRobin) NextBackend() (string, bool) {
	backends := rr.pool.HealthyBackends()
	if len(backends) == 0 {
		return "", false
	}
	idx := atomic.AddUint64(&rr.counter, 1)
	return backends[(idx-1)%uint64(len(backends))], true
}
