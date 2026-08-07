package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

// Solver is the one interface the pipeline depends on. Everything else is
// swappable behind it: a local simulation for load testing the control plane,
// or a real Crawlbase-backed solver for the end-to-end path. The pipeline never
// knows which one it is driving.
type Solver interface {
	Solve(ctx context.Context, c Challenge) error
	Name() string
}

// MockSolver simulates a solve stage: a base latency plus jitter, and a small
// failure rate. No network. This is what you use to answer "how fast is my
// orchestration, independent of the network?" -- which is the question behind an
// 8k/sec target.
type MockSolver struct {
	Base     time.Duration
	Jitter   time.Duration
	FailRate float64
}

func (s *MockSolver) Name() string { return "mock" }

func (s *MockSolver) Solve(ctx context.Context, _ Challenge) error {
	d := s.Base
	if s.Jitter > 0 {
		// The global rand source is safe for concurrent use.
		d += time.Duration(rand.Int63n(int64(s.Jitter)))
	}
	select {
	case <-time.After(d):
	case <-ctx.Done():
		return ctx.Err()
	}
	if s.FailRate > 0 && rand.Float64() < s.FailRate {
		return errors.New("mock solve failed")
	}
	return nil
}

// CrawlbaseSolver represents the real path: the CAPTCHA/anti-bot work is done by
// Crawlbase during the fetch, so "solving" here means issuing a Crawling API
// request and confirming it came back clean. This is the honest end-to-end test
// against a live endpoint; it is rate-limited by the network, not by us.
type CrawlbaseSolver struct {
	Token  string
	Client *http.Client
}

func NewCrawlbaseSolver(token string) *CrawlbaseSolver {
	return &CrawlbaseSolver{
		Token:  token,
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *CrawlbaseSolver) Name() string { return "crawlbase" }

func (s *CrawlbaseSolver) Solve(ctx context.Context, c Challenge) error {
	endpoint := fmt.Sprintf(
		"https://api.crawlbase.com/?token=%s&url=%s",
		url.QueryEscape(s.Token), url.QueryEscape(c.URL),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("crawlbase status %d", resp.StatusCode)
	}
	return nil
}
