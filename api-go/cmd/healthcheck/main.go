package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:8080/api/health")
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		os.Exit(0)
	}
	os.Exit(1)
}
