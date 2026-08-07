# CLAUDE.md — Supporting Code

Applies to `code/`. Also follow the repository-root `CLAUDE.md`.

## Purpose

Runnable Go implementation of a CAPTCHA-solve pipeline: bounded queue, worker
pool with backpressure, pluggable solver, and a metrics collector.

## Organization

```text
code/
├── README.md
├── LICENSE
├── steps/
│   ├── 01-queue-and-workers/
│   ├── 02-solver-interface/
│   ├── 03-metrics/
│   └── 04-load-harness/
└── final/
    ├── go.mod
    ├── .env.example
    ├── config.go
    ├── pipeline.go
    ├── solver.go
    ├── metrics.go
    └── main.go
```

`final/` is the canonical module. `steps/` are read-only checkpoints.

## Standards

- Go 1.22+, standard library only (no third-party dependencies).
- `Solver` is the single interface the pipeline depends on; keep the mock and
  Crawlbase implementations behind it.
- The bounded channel is the backpressure mechanism; do not make it unbounded.
- Secrets via env / `.env`; ship `.env.example` placeholders only. Never commit a
  token.
- Comments explain reasoning and Crawlbase-specific behavior, not obvious syntax.

## Crawlbase Integration

`CrawlbaseSolver` issues a Crawling API GET
(`https://api.crawlbase.com/?token=TOKEN&url=...`) and treats a non-200 as a
failed solve. The CAPTCHA/anti-bot work is performed by Crawlbase during that
fetch. Verify parameters at `https://crawlbase.com/docs` before changing them.
