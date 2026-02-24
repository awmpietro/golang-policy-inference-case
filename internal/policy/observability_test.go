package policy

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type spyNodeLatencyObserver struct {
	mu      sync.Mutex
	records []string
}

func (s *spyNodeLatencyObserver) ObserveNodeLatency(nodeID string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, nodeID)
}

func (s *spyNodeLatencyObserver) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.records)
}

func TestAsyncNodeLatencyObserver_DeliversEventsOnClose(t *testing.T) {
	spy := &spyNodeLatencyObserver{}
	async := NewAsyncNodeLatencyObserver(spy, 8)

	async.ObserveNodeLatency("start", 1*time.Millisecond)
	async.ObserveNodeLatency("approved", 2*time.Millisecond)
	async.Close()

	if got := spy.Count(); got != 2 {
		t.Fatalf("expected 2 delivered events, got %d", got)
	}
}

func TestAsyncNodeLatencyObserver_DropsWhenBufferIsFull(t *testing.T) {
	spy := &spyNodeLatencyObserver{}
	async := NewAsyncNodeLatencyObserver(spy, 1)

	for i := 0; i < 1000; i++ {
		async.ObserveNodeLatency("n", time.Microsecond)
	}
	async.Close()

	if async.Dropped() == 0 {
		t.Fatalf("expected dropped events > 0")
	}
}

func TestAsyncNodeLatencyObserver_CloseDuringConcurrentObserveDoesNotPanic(t *testing.T) {
	spy := &spyNodeLatencyObserver{}
	async := NewAsyncNodeLatencyObserver(spy, 32)

	const workers = 8
	const perWorker = 200
	var wg sync.WaitGroup
	var panics atomic.Int32

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if recover() != nil {
					panics.Add(1)
				}
			}()
			for j := 0; j < perWorker; j++ {
				async.ObserveNodeLatency("n", time.Microsecond)
			}
		}()
	}

	time.Sleep(1 * time.Millisecond)
	async.Close()
	wg.Wait()

	if panics.Load() != 0 {
		t.Fatalf("expected no panics, got %d", panics.Load())
	}
}
