package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

func getLocalIP() string {
	// try to get a non-loopback IP for extra clarity (best-effort)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "unknown"
}

func main() {
	// CLI arg or PORT env or default
	var argPort string
	flag.StringVar(&argPort, "port", "", "port to listen on (e.g. :8081)")
	flag.Parse()

	port := argPort
	if port == "" {
		if p := os.Getenv("PORT"); p != "" {
			port = p
		} else {
			port = ":8081"
		}
	}

	hostname, _ := os.Hostname()
	localIP := getLocalIP()

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from backend %s (host=%s ip=%s)\n", port, hostname, localIP)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	log.Printf("✅ Backend server running on %s (host=%s ip=%s)\n", port, hostname, localIP)
	log.Fatal(http.ListenAndServe(port, nil))
}
