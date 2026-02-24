package policy

import (
	"log"
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
