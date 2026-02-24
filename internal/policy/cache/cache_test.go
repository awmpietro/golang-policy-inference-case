package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
)

func TestInMemory_GetOrCompute_DeduplicatesConcurrentSameKey(t *testing.T) {
	c := NewInMemory(16)
	var calls atomic.Int32

	fn := func() (*policy.Policy, error) {
		calls.Add(1)
		time.Sleep(30 * time.Millisecond)
		return &policy.Policy{Start: "start", Nodes: map[string]*policy.Node{"start": {ID: "start"}}}, nil
	}

	const n = 20
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.GetOrCompute("same-key", fn)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("expected fn to run once, got %d", got)
	}
}

func TestInMemory_GetOrCompute_ErrorIsNotCached(t *testing.T) {
	c := NewInMemory(16)
	var calls atomic.Int32

	_, err := c.GetOrCompute("k", func() (*policy.Policy, error) {
		calls.Add(1)
		return nil, errors.New("boom")
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	_, err = c.GetOrCompute("k", func() (*policy.Policy, error) {
		calls.Add(1)
		return &policy.Policy{Start: "start", Nodes: map[string]*policy.Node{"start": {ID: "start"}}}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := calls.Load(); got != 2 {
		t.Fatalf("expected fn to run twice (error should not be cached), got %d", got)
	}
}

func TestInMemory_GetOrCompute_PanicDoesNotBlockWaiters(t *testing.T) {
	c := NewInMemory(16)
	var calls atomic.Int32

	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.GetOrCompute("panic-key", func() (*policy.Policy, error) {
				calls.Add(1)
				time.Sleep(10 * time.Millisecond)
				panic("boom")
			})
			errs <- err
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err == nil {
			t.Fatalf("expected panic converted into error")
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected single in-flight execution, got %d", got)
	}
}
