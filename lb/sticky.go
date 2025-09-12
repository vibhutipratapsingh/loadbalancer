package lb

import (
	"crypto/sha1"
	"net"
	"sync"
	"time"
)

// StickyMap stores clientID -> backend mapping (optional; useful for cookie revocation or debug)
type StickyMap struct {
	mu    sync.RWMutex
	store map[string]string // clientID -> backendAddr
	ttl   time.Duration
	// Note: for production use Redis or other external store if you have multiple LB instances.
}

func NewStickyMap(ttl time.Duration) *StickyMap {
	return &StickyMap{
		store: make(map[string]string),
		ttl:   ttl,
	}
}

func (s *StickyMap) Get(clientID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.store[clientID]
	return b, ok
}

func (s *StickyMap) Set(clientID, backend string) {
	s.mu.Lock()
	s.store[clientID] = backend
	s.mu.Unlock()
	// TTL eviction omitted for brevity; add background janitor if needed.
}

func (s *StickyMap) Delete(clientID string) {
	s.mu.Lock()
	delete(s.store, clientID)
	s.mu.Unlock()
}

// ---------------- IP-HASH STICKY ----------------

// ipToIndex deterministically converts an IP (string) to an index in [0, n-1].
// Returns -1 if IP parsing fails or n==0.
func ipToIndex(clientIP string, n int) int {
	if n <= 0 {
		return -1
	}
	ip := net.ParseIP(clientIP)
	if ip == nil {
		// fallback: hash string
		h := sha1.Sum([]byte(clientIP))
		return int(h[0]) % n
	}
	// use last 4 bytes for IPv4, or mix for IPv6
	ip4 := ip.To4()
	if ip4 != nil {
		// simple mixing of 4 bytes
		sum := int(ip4[0]) + int(ip4[1]) + int(ip4[2]) + int(ip4[3])
		return sum % n
	}
	// IPv6 fallback
	h := sha1.Sum([]byte(ip.String()))
	return int(h[0]) % n
}
