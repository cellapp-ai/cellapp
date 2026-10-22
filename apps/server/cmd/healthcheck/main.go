package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	client := http.Client{Timeout: 3 * time.Second}
	for _, probe := range []struct {
		url string
		status int
	}{
		{"http://127.0.0.1:8080/health", http.StatusOK},
		{"http://127.0.0.1:3001/", 0},
	} {
		resp, err := client.Get(probe.url)
		if err != nil {
			os.Exit(1)
		}
		resp.Body.Close()
		if probe.status != 0 && resp.StatusCode != probe.status || resp.StatusCode >= 500 {
			os.Exit(1)
		}
	}
}
