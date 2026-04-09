package api

import "net/http"
import "time"

type Client struct {
	base       string
	httpClient *http.Client
}

func NewClient(base string) *Client {
	return &Client{base: base, httpClient: &http.Client{Timeout: 10 * time.Second}}
}
