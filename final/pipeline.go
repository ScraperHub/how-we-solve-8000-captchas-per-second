package main

import (
	"context"
	"sync"
	"time"
)

// Challenge is one unit of work. In a real system it carries the site key and
// the page it belongs to; here URL is what the Crawlbase solver actually fetches.
type Challenge struct {
	ID      int
	URL     string
	SiteKey string
}

// Result is what a worker reports for each challenge.
type Result struct {
	ID      int
	OK      bool
	Latency time.Duration
	Worker  int
}

// Pipeline is the control plane: a bounded queue, a pool of workers, and a
// results channel that fans in to the metrics collector. The bound on the queue
// is the backpressure mechanism -- when producers outrun workers, the send
// blocks instead of letting memory grow without limit.
type Pipeline struct {
	workers int
	queue   chan Challenge
	results chan Result
	solver  Solver
	metrics *Metrics
	target  string
}

func NewPipeline(workers, queueSize int, solver Solver, metrics *Metrics, target string) *Pipeline {
	return &Pipeline{
		workers: workers,
		queue:   make(chan Challenge, queueSize), // bounded => backpressure
		results: make(chan Result, queueSize),
		solver:  solver,
		metrics: metrics,
		target:  target,
	}
}

// Run drives `total` challenges through the pipeline and returns the report.
func (p *Pipeline) Run(ctx context.Context, total int) Report {
	var workers sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		workers.Add(1)
		go p.worker(ctx, i, &workers)
	}

	// Fan-in: a single collector drains results into metrics. Keeping this on
	// one goroutine means the metrics struct is only written from here plus its
	// own mutex, which keeps the hot path simple.
	collectorDone := make(chan struct{})
	go func() {
		for r := range p.results {
			p.metrics.Record(r.Latency, r.OK)
		}
		close(collectorDone)
	}()

	// Producer: enqueue work, respecting cancellation. Closing the queue is the
	// signal that lets workers finish.
	p.metrics.Start()
	go func() {
		defer close(p.queue)
		for i := 0; i < total; i++ {
			ch := Challenge{ID: i, URL: p.target, SiteKey: "demo-site-key"}
			select {
			case p.queue <- ch:
			case <-ctx.Done():
				return
			}
		}
	}()

	workers.Wait()      // workers exit once the queue is drained and closed
	close(p.results)    // no more results will be sent
	<-collectorDone     // wait for the collector to finish recording
	p.metrics.Stop()
	return p.metrics.Report()
}

func (p *Pipeline) worker(ctx context.Context, id int, wg *sync.WaitGroup) {
	defer wg.Done()
	for ch := range p.queue {
		started := time.Now()
		err := p.solver.Solve(ctx, ch)
		p.results <- Result{
			ID:      ch.ID,
			OK:      err == nil,
			Latency: time.Since(started),
			Worker:  id,
		}
	}
}
