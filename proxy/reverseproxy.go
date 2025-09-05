package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// ForwardRequest forwards a request to the given backend server
func ForwardRequest(backend string, w http.ResponseWriter, r *http.Request) error {
	target, err := url.Parse(backend)
	if err != nil {
		return fmt.Errorf("invalid backend URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Log request → backend mapping
	fmt.Printf("Request %s → %s\n", r.URL.Path, backend)

	proxy.ServeHTTP(w, r)
	return nil
}
