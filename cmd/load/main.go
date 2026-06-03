package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	url := flag.String("url", "http://localhost:8080/order?id=order-1", "request URL")
	concurrency := flag.Int("c", 20, "number of concurrent workers")
	requests := flag.Int("n", 10000, "total number of requests")
	timeout := flag.Duration("timeout", 5*time.Second, "request timeout")

	flag.Parse()

	if *concurrency <= 0 {
		log.Fatal("concurrency must be greater than 0")
	}

	if *requests <= 0 {
		log.Fatal("requests count must be greater than 0")
	}

	client := &http.Client{
		Timeout: *timeout,
	}

	var okCount int64
	var errorCount int64
	var totalLatencyNs int64

	jobs := make(chan int)

	start := time.Now()

	var wg sync.WaitGroup

	for i := 0; i < *concurrency; i++ {
		wg.Go(func() {

			for range jobs {
				requestStart := time.Now()

				resp, err := client.Get(*url)
				latency := time.Since(requestStart)

				atomic.AddInt64(&totalLatencyNs, latency.Nanoseconds())

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()

				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					atomic.AddInt64(&okCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		})

	}

	for i := 0; i < *requests; i++ {
		jobs <- i
	}

	close(jobs)
	wg.Wait()

	duration := time.Since(start)

	ok := atomic.LoadInt64(&okCount)
	errors := atomic.LoadInt64(&errorCount)
	totalLatency := atomic.LoadInt64(&totalLatencyNs)

	var avgLatency time.Duration
	if ok+errors > 0 {
		avgLatency = time.Duration(totalLatency / (ok + errors))
	}

	rps := float64(ok+errors) / duration.Seconds()

	fmt.Println("Load test finished")
	fmt.Printf("URL:          %s\n", *url)
	fmt.Printf("Requests:     %d\n", *requests)
	fmt.Printf("Concurrency:  %d\n", *concurrency)
	fmt.Printf("OK:           %d\n", ok)
	fmt.Printf("Errors:       %d\n", errors)
	fmt.Printf("Duration:     %s\n", duration)
	fmt.Printf("RPS:          %.2f\n", rps)
	fmt.Printf("Avg latency:  %s\n", avgLatency)
}
