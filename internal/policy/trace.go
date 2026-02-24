package policy

type ExecutionTrace struct {
	StartNode   string      `json:"start_node"`
	VisitedPath []string    `json:"visited_path"`
	Steps       []TraceStep `json:"steps"`
	Terminated  string      `json:"terminated"`
}

type TraceStep struct {
	NodeID         string      `json:"node_id"`
	DurationMicros int64       `json:"duration_micros"`
	ChosenNext     string      `json:"chosen_next,omitempty"`
	Edges          []EdgeTrace `json:"edges,omitempty"`
}

type EdgeTrace struct {
	To      string `json:"to"`
	Cond    string `json:"cond"`
	Matched bool   `json:"matched"`
	Error   string `json:"error,omitempty"`
}
