package agent

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type HttpClient struct {
	mu        sync.Mutex
	client    *http.Client
	transport *http.Transport
}

func NewHttpClient() *HttpClient {
	transport := newTransport()
	return &HttpClient{
		client:    &http.Client{Transport: transport},
		transport: transport,
	}
}

func newTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 40,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   false,
	}
}

// Start pre-warms TLS connections and keeps them alive with periodic pings.
func (c *HttpClient) Start(ctx context.Context) {
	t := time.NewTicker(10 * time.Second)
	for i := 0; i < 20; i++ {
		c.doPingRequest()
		time.Sleep(100 * time.Millisecond)
	}
	for {
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
			c.doPingRequest()
		}
	}
}

func (c *HttpClient) doPingRequest() {
	resp, err := c.client.Get("https://" + hostName + pingEndpoint)
	if err != nil {
		slog.Error("failed to send ping request", "error", err, "host", hostName)
		return
	}
	// drain body to allow connection reuse
	_, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		slog.Error("failed to read ping response body", "error", err, "host", hostName)
	}
}

// Do executes an HTTP request. On connection reset it recreates the transport.
func (c *HttpClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil && isConnectionReset(err) {
		slog.Info("connection reset detected, recreating transport")
		c.recreateTransport()
	}
	return resp, err
}

func isConnectionReset(err error) bool {
	if err == nil {
		return false
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Op == "read" && strings.Contains(netErr.Err.Error(), "connection reset by peer") {
			return true
		}
	}
	return strings.Contains(err.Error(), "connection reset by peer")
}

func (c *HttpClient) recreateTransport() {
	c.mu.Lock()
	defer c.mu.Unlock()
	old := c.transport
	c.transport = newTransport()
	c.client.Transport = c.transport
	old.CloseIdleConnections()
}
