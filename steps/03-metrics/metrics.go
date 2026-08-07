package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Metrics collects per-solve latencies and the success/failure counts, then
// turns them into the numbers that actually matter for a throughput system:
// solves per second and the tail latency (p99). The mutex keeps it safe when
// every worker reports concurrently.
type Metrics struct {
	mu        sync.Mutex
	latencies []time.Duration
	successes int
	failures  int
	start     time.Time
	end       time.Time
}

func NewMetrics(capacity int) *Metrics {
	return &Metrics{latencies: make([]time.Duration, 0, capacity)}
}

func (m *Metrics) Start() { m.start = time.Now() }
func (m *Metrics) Stop()  { m.end = time.Now() }

func (m *Metrics) Record(latency time.Duration, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latencies = append(m.latencies, latency)
	if ok {
		m.successes++
	} else {
		m.failures++
	}
}

func (m *Metrics) percentile(p float64) time.Duration {
	if len(m.latencies) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(m.latencies))
	copy(sorted, m.latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(p/100*float64(len(sorted)-1) + 0.5)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// Report is the printable summary. SolvesPerSecond is the headline number; the
// percentiles tell you whether that number hides a bad tail.
type Report struct {
	Total           int
	Successes       int
	Failures        int
	Elapsed         time.Duration
	SolvesPerSecond float64
	P50             time.Duration
	P99             time.Duration
}

func (m *Metrics) Report() Report {
	total := m.successes + m.failures
	elapsed := m.end.Sub(m.start)
	rate := 0.0
	if elapsed > 0 {
		rate = float64(total) / elapsed.Seconds()
	}
	return Report{
		Total:           total,
		Successes:       m.successes,
		Failures:        m.failures,
		Elapsed:         elapsed,
		SolvesPerSecond: rate,
		P50:             m.percentile(50),
		P99:             m.percentile(99),
	}
}

func (r Report) String() string {
	return fmt.Sprintf(
		"total=%d  ok=%d  fail=%d  elapsed=%s\nsolves/sec=%.0f  p50=%s  p99=%s",
		r.Total, r.Successes, r.Failures, r.Elapsed.Round(time.Millisecond),
		r.SolvesPerSecond, r.P50.Round(time.Microsecond), r.P99.Round(time.Microsecond),
	)
}
