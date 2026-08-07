// Command captcha-pipeline runs a CAPTCHA-solve pipeline and reports throughput.
//
// The mock solver load-tests the control plane (queue + workers + fan-in) with
// no network, so you can see how many solves/sec the orchestration sustains on
// one machine. The crawlbase solver runs the real end-to-end path, where the
// CAPTCHA and anti-bot work is handled by Crawlbase during the fetch.
//
// Usage:
//
//	go run . -requests 20000 -workers 256 -queue 1024              # mock
//	go run . -solver crawlbase -requests 20 -workers 8             # real path
package main

import (
	"context"
	"flag"
	"fmt"
	"time"
)

func main() {
	solverName := flag.String("solver", "mock", "solver to use: mock | crawlbase")
	requests := flag.Int("requests", 20000, "number of challenges to process")
	workers := flag.Int("workers", 256, "number of concurrent workers")
	queueSize := flag.Int("queue", 1024, "bounded queue size (backpressure)")
	baseMs := flag.Int("base-ms", 5, "mock: base solve latency in ms")
	jitterMs := flag.Int("jitter-ms", 5, "mock: added random latency in ms")
	failRate := flag.Float64("fail-rate", 0.01, "mock: fraction of solves that fail")
	flag.Parse()

	cfg := loadConfig()

	var solver Solver
	switch *solverName {
	case "crawlbase":
		if cfg.CrawlbaseToken == "" {
			fmt.Println("error: CRAWLBASE_TOKEN is required for the crawlbase solver")
			fmt.Println("set it in .env or the environment, or use -solver mock")
			return
		}
		solver = NewCrawlbaseSolver(cfg.CrawlbaseToken)
	default:
		solver = &MockSolver{
			Base:     time.Duration(*baseMs) * time.Millisecond,
			Jitter:   time.Duration(*jitterMs) * time.Millisecond,
			FailRate: *failRate,
		}
	}

	fmt.Printf("solver=%s requests=%d workers=%d queue=%d target=%s\n\n",
		solver.Name(), *requests, *workers, *queueSize, cfg.TargetURL)

	metrics := NewMetrics(*requests)
	pipeline := NewPipeline(*workers, *queueSize, solver, metrics, cfg.TargetURL)

	report := pipeline.Run(context.Background(), *requests)
	fmt.Println(report)
}
