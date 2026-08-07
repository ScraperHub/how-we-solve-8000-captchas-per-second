package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Config holds everything the pipeline needs. We keep it tiny and read it from
// the environment, with a zero-dependency .env loader so you can drop a file in
// during local runs. No secrets live in the source.
type Config struct {
	CrawlbaseToken string
	TargetURL      string
}

// loadDotEnv reads a .env file if one exists next to the binary's working dir.
// It is intentionally minimal: KEY=VALUE per line, # for comments. We do not
// pull in a dependency for something this small.
func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return // no .env is fine; env vars may be set another way
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}

func loadConfig() Config {
	loadDotEnv(".env")
	target := os.Getenv("TARGET_URL")
	if target == "" {
		target = "https://example.com"
	}
	return Config{
		CrawlbaseToken: os.Getenv("CRAWLBASE_TOKEN"),
		TargetURL:      target,
	}
}

func envInt(name string, fallback int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
