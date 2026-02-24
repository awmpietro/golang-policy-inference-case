package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
)

type InMemory struct {
	mu    sync.RWMutex
	max   int
	items map[string]*policy.Policy
}

func NewInMemory(max int) *InMemory {
	return &InMemory{
		max:   max,
		items: make(map[string]*policy.Policy, max),
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
	defer c.mu.Unlock()

	if v, ok := c.items[key]; ok {
		return v, nil
	}

	p, err := fn()
	if err != nil {
		return nil, err
	}

	if len(c.items) < c.max {
		c.items[key] = p
	}

	return p, nil
}

func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
