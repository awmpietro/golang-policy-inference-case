package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

type inferPayload struct {
	PolicyDOT string         `json:"policy_dot"`
	Input     map[string]any `json:"input"`
}

type result struct {
	latency time.Duration
	status  int
	err     error
}

func main() {
	url := flag.String("url", "http://localhost:8080/infer", "infer endpoint URL")
	rps := flag.Int("rps", 50, "target requests per second")
	duration := flag.Duration("duration", 60*time.Second, "test duration")
	workers := flag.Int("workers", 50, "number of concurrent workers")
	timeout := flag.Duration("timeout", 5*time.Second, "HTTP client timeout")
	flag.Parse()

	if *rps <= 0 || *duration <= 0 || *workers <= 0 {
		fmt.Fprintln(os.Stderr, "rps, duration and workers must be > 0")
		os.Exit(2)
	}

	payload := inferPayload{
		PolicyDOT: `digraph Policy {
			start [result=""]
			approved [result="approved=true,segment=prime"]
			review [result="approved=false,segment=manual"]
			rejected [result="approved=false"]
			start -> approved [cond="age>=18 && score>700"]
			start -> review [cond="age>=18 && score<=700"]
			start -> rejected [cond="age<18"]
		}`,
		Input: map[string]any{"age": 25, "score": 720},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal payload: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: *timeout}
	jobs := make(chan struct{}, *workers)

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]result, 0, *rps*int(duration.Seconds())+1)

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				start := time.Now()
				req, err := http.NewRequest(http.MethodPost, *url, bytes.NewReader(body))
				if err != nil {
					mu.Lock()
					results = append(results, result{latency: time.Since(start), err: err})
					mu.Unlock()
					continue
				}
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				lat := time.Since(start)
				if err != nil {
					mu.Lock()
					results = append(results, result{latency: lat, err: err})
					mu.Unlock()
					continue
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				mu.Lock()
				results = append(results, result{latency: lat, status: resp.StatusCode})
				mu.Unlock()
			}
		}()
	}

	interval := time.Second / time.Duration(*rps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	deadline := time.Now().Add(*duration)
	launched := 0

	for now := range ticker.C {
		if now.After(deadline) {
			break
		}
		jobs <- struct{}{}
		launched++
	}
	close(jobs)
	wg.Wait()

	latencies := make([]time.Duration, 0, len(results))
	success2xx := 0
	non2xx := 0
	errs := 0

	for _, r := range results {
		latencies = append(latencies, r.latency)
		if r.err != nil {
			errs++
			continue
		}
		if r.status >= 200 && r.status < 300 {
			success2xx++
		} else {
			non2xx++
		}
	}

	if len(latencies) == 0 {
		fmt.Fprintln(os.Stderr, "no requests executed")
		os.Exit(1)
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p50 := percentile(latencies, 50)
	p90 := percentile(latencies, 90)
	p99 := percentile(latencies, 99)
	avg := average(latencies)
	achievedRPS := float64(len(latencies)) / duration.Seconds()

	fmt.Printf("Load test finished\n")
	fmt.Printf("- target_rps: %d\n", *rps)
	fmt.Printf("- achieved_rps: %.2f\n", achievedRPS)
	fmt.Printf("- duration: %s\n", duration.String())
	fmt.Printf("- requests: %d\n", len(latencies))
	fmt.Printf("- 2xx: %d\n", success2xx)
	fmt.Printf("- non_2xx: %d\n", non2xx)
	fmt.Printf("- errors: %d\n", errs)
	fmt.Printf("- avg_ms: %.3f\n", ms(avg))
	fmt.Printf("- p50_ms: %.3f\n", ms(p50))
	fmt.Printf("- p90_ms: %.3f\n", ms(p90))
	fmt.Printf("- p99_ms: %.3f\n", ms(p99))

	minRPS := float64(*rps) * 0.98
	if achievedRPS >= minRPS && p90 < 30*time.Millisecond && errs == 0 && non2xx == 0 {
		fmt.Println("PASS: meets 50 RPS and P90 < 30ms")
		return
	}

	fmt.Println("FAIL: does not meet target (or has request errors)")
	os.Exit(1)
}

func percentile(items []time.Duration, p int) time.Duration {
	if len(items) == 0 {
		return 0
	}
	idx := (len(items) - 1) * p / 100
	return items[idx]
}

func average(items []time.Duration) time.Duration {
	if len(items) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range items {
		total += d
	}
	return total / time.Duration(len(items))
}

func ms(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000.0
}
