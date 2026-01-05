// Package client provides HTTP clients for calling ML model services.
package client

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/playground/apps/guardrail-server-go/internal/config"
)

// Pool manages HTTP clients for model services.
type Pool struct {
	clients map[string]*http.Client
	mu      sync.RWMutex
	cfg     *config.Config
}

// NewPool creates a new client pool.
func NewPool(cfg *config.Config) *Pool {
	return &Pool{
		clients: make(map[string]*http.Client),
		cfg:     cfg,
	}
}

// Get returns an HTTP client for the specified model.
func (p *Pool) Get(modelName string) *http.Client {
	p.mu.RLock()
	client, exists := p.clients[modelName]
	p.mu.RUnlock()

	if exists {
		return client
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists = p.clients[modelName]; exists {
		return client
	}

	client = p.createClient()
	p.clients[modelName] = client
	return client
}

// createClient creates a new HTTP client with configured timeouts.
func (p *Pool) createClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   p.cfg.ModelConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: p.cfg.ModelTimeout,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   p.cfg.ModelTimeout + p.cfg.ModelConnectTimeout,
	}
}

// GetBaseURL returns the base URL for a model.
func (p *Pool) GetBaseURL(modelName string) string {
	urls := p.cfg.ModelURLs()
	return urls[modelName]
}

// CloseAll closes all clients in the pool.
func (p *Pool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, client := range p.clients {
		client.CloseIdleConnections()
	}
	p.clients = make(map[string]*http.Client)
}
