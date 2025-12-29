package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Runner orchestrates the load test execution
type Runner struct {
	cfg     Config
	client  *Client
	metrics *Metrics
}

// NewRunner creates a new load test runner
func NewRunner(cfg Config, client *Client) *Runner {
	return &Runner{
		cfg:     cfg,
		client:  client,
		metrics: NewMetrics(),
	}
}

// Run executes the load test
func (r *Runner) Run(ctx context.Context) *Results {
	// Channel for work distribution
	requestCh := make(chan struct{}, r.cfg.Workers*2)
	
	// Wait group for workers
	var wg sync.WaitGroup

	// Start metrics collection
	r.metrics.Start()

	// Start workers
	for i := 0; i < r.cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			r.worker(ctx, workerID, requestCh)
		}(i)
	}

	// Rate limiter: send requests at target RPS
	ticker := time.NewTicker(time.Second / time.Duration(r.cfg.RPS))
	defer ticker.Stop()

	// Progress reporting
	progressTicker := time.NewTicker(5 * time.Second)
	defer progressTicker.Stop()

	startTime := time.Now()
	requestsSent := 0

	fmt.Println("🚀 Load test started...")
	fmt.Println()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop

		case <-progressTicker.C:
			elapsed := time.Since(startTime)
			currentRPS := float64(requestsSent) / elapsed.Seconds()
			remaining := r.cfg.Duration - elapsed
			fmt.Printf("⏱️  Elapsed: %s | Remaining: %s | Requests: %d | Current RPS: %.1f\n",
				formatDuration(elapsed),
				formatDuration(remaining),
				requestsSent,
				currentRPS)

		case <-ticker.C:
			select {
			case requestCh <- struct{}{}:
				requestsSent++
			default:
				// Channel full, workers are falling behind
			}
		}
	}

	// Close request channel and wait for workers
	close(requestCh)
	wg.Wait()

	// Stop metrics collection
	r.metrics.Stop()

	fmt.Println()
	fmt.Println("✅ Load test completed!")

	return r.metrics.GetResults(r.cfg.RPS)
}

// worker processes requests from the request channel
func (r *Runner) worker(ctx context.Context, id int, requestCh <-chan struct{}) {
	for range requestCh {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Create request context with timeout
		reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		result := r.client.SendRequest(reqCtx)
		cancel()

		// Record the result
		r.metrics.Record(result)
	}
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "0s"
	}
	d = d.Round(time.Second)
	m := d / time.Minute
	s := (d % time.Minute) / time.Second
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
