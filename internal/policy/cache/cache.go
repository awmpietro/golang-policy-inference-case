package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
)

type InMemory struct {
	mu       sync.RWMutex
	max      int
	items    map[string]*policy.Policy
	inflight map[string]*inflightCall
}

type inflightCall struct {
	wg  sync.WaitGroup
	p   *policy.Policy
	err error
}

func NewInMemory(max int) *InMemory {
	return &InMemory{
		max:      max,
		items:    make(map[string]*policy.Policy, max),
		inflight: make(map[string]*inflightCall),
	}
}

func (c *InMemory) GetOrCompute(dot string, fn func() (*policy.Policy, error)) (*policy.Policy, error) {
	key := hash(dot)

	c.mu.RLock()
	if v, ok := c.items[key]; ok {
		c.mu.RUnlock()
		return v, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	if v, ok := c.items[key]; ok {
		c.mu.Unlock()
		return v, nil
	}

	if call, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		call.wg.Wait()
		return call.p, call.err
	}

	call := &inflightCall{}
	call.wg.Add(1)
	c.inflight[key] = call
	c.mu.Unlock()

	func() {
		defer func() {
			if r := recover(); r != nil {
				call.err = fmt.Errorf("compute panic: %v", r)
			}
			c.mu.Lock()
			delete(c.inflight, key)
			if call.err == nil && call.p != nil && len(c.items) < c.max {
				c.items[key] = call.p
			}
			c.mu.Unlock()
			call.wg.Done()
		}()
		call.p, call.err = fn()
	}()

	return call.p, call.err
}

func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
