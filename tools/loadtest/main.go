// Package main provides a load testing tool for the Guardrails platform.
// It simulates multiple tenants calling the validation API and collects
// detailed latency metrics for performance analysis.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Config holds the load test configuration
type Config struct {
	Target   string        // Guardrail API URL
	Duration time.Duration // Test duration
	RPS      int           // Target requests per second
	Workers  int           // Number of concurrent workers
	Tenants  int           // Number of simulated tenants
	Output   string        // Output format (json/text)
}

func main() {
	cfg := parseFlags()

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              GUARDRAIL LOAD TEST                              ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Target       : %-45s ║\n", cfg.Target)
	fmt.Printf("║  Duration     : %-45s ║\n", cfg.Duration)
	fmt.Printf("║  Target RPS   : %-45d ║\n", cfg.RPS)
	fmt.Printf("║  Workers      : %-45d ║\n", cfg.Workers)
	fmt.Printf("║  Tenants      : %-45d ║\n", cfg.Tenants)
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create context with cancellation
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n⚠️  Received shutdown signal, finishing current requests...")
		cancel()
	}()

	// Create client and runner
	client := NewClient(cfg.Target, cfg.Tenants)
	runner := NewRunner(cfg, client)

	// Run the load test
	results := runner.Run(ctx)

	// Print results
	if cfg.Output == "json" {
		printJSONResults(results)
	} else {
		printTextResults(results)
	}
}

func parseFlags() Config {
	cfg := Config{}

	flag.StringVar(&cfg.Target, "target", "http://localhost:8000", "Guardrail API URL")
	flag.DurationVar(&cfg.Duration, "duration", 60*time.Second, "Test duration (e.g., 60s, 5m)")
	flag.IntVar(&cfg.RPS, "rps", 100, "Target requests per second")
	flag.IntVar(&cfg.Workers, "workers", 10, "Number of concurrent workers")
	flag.IntVar(&cfg.Tenants, "tenants", 5, "Number of simulated tenants")
	flag.StringVar(&cfg.Output, "output", "text", "Output format (json/text)")

	flag.Parse()

	// Validate
	if cfg.RPS < 1 {
		cfg.RPS = 1
	}
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}
	if cfg.Tenants < 1 {
		cfg.Tenants = 1
	}

	return cfg
}

func printTextResults(results *Results) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    LOAD TEST RESULTS                          ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Duration        : %-42.1fs ║\n", results.Duration.Seconds())
	fmt.Printf("║  Target RPS      : %-42d ║\n", results.TargetRPS)
	fmt.Printf("║  Achieved RPS    : %-42.1f ║\n", results.AchievedRPS)
	fmt.Printf("║  Total Requests  : %-42s ║\n", formatNumber(results.TotalRequests))
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  LATENCY (ms)                                                 ║")
	fmt.Printf("║    P50           : %-42.1f ║\n", results.LatencyP50)
	fmt.Printf("║    P90           : %-42.1f ║\n", results.LatencyP90)
	fmt.Printf("║    P95           : %-42.1f ║\n", results.LatencyP95)
	fmt.Printf("║    P99           : %-42.1f ║\n", results.LatencyP99)
	fmt.Printf("║    Max           : %-42.1f ║\n", results.LatencyMax)
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  SUCCESS/ERROR                                                ║")
	fmt.Printf("║    Success       : %-7s (%.1f%%)                             ║\n",
		formatNumber(results.SuccessCount),
		float64(results.SuccessCount)/float64(results.TotalRequests)*100)
	fmt.Printf("║    Timeout       : %-7s (%.1f%%)                             ║\n",
		formatNumber(results.TimeoutCount),
		float64(results.TimeoutCount)/float64(max(results.TotalRequests, 1))*100)
	fmt.Printf("║    Server Error  : %-7s (%.1f%%)                             ║\n",
		formatNumber(results.ErrorCount),
		float64(results.ErrorCount)/float64(max(results.TotalRequests, 1))*100)
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")

	// Print per-tenant breakdown if we have multiple tenants
	if len(results.TenantResults) > 1 {
		fmt.Println()
		fmt.Println("Per-Tenant Breakdown:")
		fmt.Println("┌────────────┬───────────┬──────────┬──────────┬──────────┐")
		fmt.Println("│ Tenant     │ Requests  │ Success  │ P50 (ms) │ P99 (ms) │")
		fmt.Println("├────────────┼───────────┼──────────┼──────────┼──────────┤")
		for _, tr := range results.TenantResults {
			fmt.Printf("│ %-10s │ %9d │ %7.1f%% │ %8.1f │ %8.1f │\n",
				tr.TenantID,
				tr.Requests,
				tr.SuccessRate*100,
				tr.LatencyP50,
				tr.LatencyP99)
		}
		fmt.Println("└────────────┴───────────┴──────────┴──────────┴──────────┘")
	}
}

func printJSONResults(results *Results) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling results: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func formatNumber(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
