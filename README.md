# CAPTCHA Pipeline (Go)

**Repository:** [github.com/ScraperHub/how-we-solve-8000-captchas-per-second](https://github.com/ScraperHub/how-we-solve-8000-captchas-per-second)

Go CAPTCHA-solve pipeline with bounded queue, worker pool, pluggable solvers,
and metrics. Shows how production architecture reaches 8,000 solves/sec.

A small, fast Go pipeline that shows how a CAPTCHA-solve control plane reaches
high throughput: a bounded queue, a worker pool with backpressure, a pluggable
solver, and a metrics collector that reports solves/sec and tail latency.

Two solvers:

- **mock** — simulates the solve stage with no network, so you can measure the
  orchestration itself. This is how you see whether the control plane can sustain
  thousands of solves per second on one machine.
- **crawlbase** — the real end-to-end path, where the CAPTCHA and anti-bot work
  is handled by Crawlbase during the fetch.

## Prerequisites

- Go 1.22 or newer.
- For the real path only: a Crawlbase account and token
  (https://crawlbase.com/signup?signup=blog).

## Installation

```bash
cd final
cp .env.example .env   # only needed for the crawlbase solver
go build -o captcha-pipeline .
```

## Environment variables

| Variable          | Purpose                                                  |
| ----------------- | ------------------------------------------------------- |
| `CRAWLBASE_TOKEN` | Token for the `-solver crawlbase` path.                 |
| `TARGET_URL`      | URL the crawlbase solver fetches (default example.com). |

## Running

```bash
# Load-test the control plane (no network):
go run . -requests 20000 -workers 256 -queue 1024

# Real end-to-end path through Crawlbase (small N; it is network-bound):
go run . -solver crawlbase -requests 12 -workers 6 -queue 32
```

Flags: `-solver`, `-requests`, `-workers`, `-queue`, `-base-ms`, `-jitter-ms`,
`-fail-rate`.

### Expected output (high level)

```
solver=mock requests=20000 workers=256 queue=1024 target=https://example.com

total=20000  ok=19804  fail=196  elapsed=615ms
solves/sec=32507  p50=7.83ms  p99=10.354ms
```

The mock run sustains tens of thousands of solves/sec on a single machine. The
crawlbase run is slower per item because it is bounded by the real network and
the actual solve, which is the point: the control plane is not the bottleneck.

## Troubleshooting

- **`CRAWLBASE_TOKEN is required for the crawlbase solver`.** Set it in `.env`
  or the environment, or use `-solver mock`.
- **Low solves/sec on mock.** Raise `-workers` and `-queue`; lower `-base-ms`.
- **Failures on mock.** Expected — `-fail-rate` defaults to 1%. Set it to 0 to
  see a clean run.

## Article-to-code map

| Article section                              | Code path                       |
| --------------------------------------------- | ------------------------------ |
| Step 1: Model the work — a queue and a pool   | `final/pipeline.go`            |
| Step 2: The solver interface — mock + Crawlbase | `final/solver.go`            |
| Step 3: Measure what matters — solves/sec, p99 | `final/metrics.go`            |
| Step 4: The load harness — turn the dials     | `final/main.go`, `final/config.go` |

## Project layout

`final/` is the canonical, runnable module. `steps/` holds read-only checkpoint
copies of the file introduced at each article step. Build and run from `final/`.

---
Copyright 2026 Crawlbase https://crawlbase.com/
