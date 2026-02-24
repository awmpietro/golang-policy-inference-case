package policy

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type NodeLatencyObserver interface {
	ObserveNodeLatency(nodeID string, duration time.Duration)
}

type NodeLatencyLogger struct {
	logger *log.Logger
}

func NewNodeLatencyLogger(logger *log.Logger) *NodeLatencyLogger {
	return &NodeLatencyLogger{logger: logger}
}

func (l *NodeLatencyLogger) ObserveNodeLatency(nodeID string, duration time.Duration) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Printf("policy_node_latency node=%s duration_ms=%.3f", nodeID, float64(duration.Microseconds())/1000.0)
}

type AsyncNodeLatencyObserver struct {
	next    NodeLatencyObserver
	events  chan nodeLatencyEvent
	once    sync.Once
	mu      sync.RWMutex
	closed  bool
	wg      sync.WaitGroup
	dropped atomic.Uint64
}

type nodeLatencyEvent struct {
	nodeID   string
	duration time.Duration
}

func NewAsyncNodeLatencyObserver(next NodeLatencyObserver, buffer int) *AsyncNodeLatencyObserver {
	if buffer <= 0 {
		buffer = 1
	}

	o := &AsyncNodeLatencyObserver{
		next:   next,
		events: make(chan nodeLatencyEvent, buffer),
	}

	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		for ev := range o.events {
			if o.next == nil {
				continue
			}
			o.next.ObserveNodeLatency(ev.nodeID, ev.duration)
		}
	}()

	return o
}

func (o *AsyncNodeLatencyObserver) ObserveNodeLatency(nodeID string, duration time.Duration) {
	if o == nil {
		return
	}
	o.mu.RLock()
	if o.closed {
		o.mu.RUnlock()
		o.dropped.Add(1)
		return
	}
	select {
	case o.events <- nodeLatencyEvent{nodeID: nodeID, duration: duration}:
	default:
		o.dropped.Add(1)
	}
	o.mu.RUnlock()
}

func (o *AsyncNodeLatencyObserver) Dropped() uint64 {
	if o == nil {
		return 0
	}
	return o.dropped.Load()
}

func (o *AsyncNodeLatencyObserver) Close() {
	if o == nil {
		return
	}
	o.once.Do(func() {
		o.mu.Lock()
		o.closed = true
		close(o.events)
		o.mu.Unlock()
		o.wg.Wait()
	})
}
