package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// createTransport returns an HTTP transport which does not reuse connections.
// This increases the chance the Docker service-name DNS will resolve to different
// container IPs per new request.
func createTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          0,
		IdleConnTimeout:       0,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     true, // <--- disable keep-alive so connections are not reused
	}
}

// ForwardRequest forwards a request to the given backend server.
func ForwardRequest(backend string, w http.ResponseWriter, r *http.Request) error {
	target, err := url.Parse(backend)
	if err != nil {
		return fmt.Errorf("invalid backend URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	// assign a fresh transport per proxy so requests don't reuse connections
	proxy.Transport = createTransport()

	// log mapping
	fmt.Printf("Request %s → %s\n", r.URL.Path, backend)

	proxy.ServeHTTP(w, r)
	return nil
}
