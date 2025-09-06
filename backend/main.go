package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Port required: go run backend/main.go :8081")
	}
	port := os.Args[1]

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from backend %s\n", port)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	fmt.Printf("✅ Backend server running on %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
