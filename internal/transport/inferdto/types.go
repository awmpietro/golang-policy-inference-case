package inferdto

import "github.com/awmpietro/golang-policy-inference-case/internal/app"

type InferRequest struct {
	PolicyDOT string         `json:"policy_dot"`
	Input     map[string]any `json:"input"`
	PolicyID  string         `json:"policy_id,omitempty"`
	Version   string         `json:"policy_version,omitempty"`
	Debug     bool           `json:"debug,omitempty"`
}

func (r InferRequest) Options() app.InferOptions {
	return app.InferOptions{
		PolicyID:      r.PolicyID,
		PolicyVersion: r.Version,
	}
}

type InferResponse struct {
	Output map[string]any  `json:"output"`
	Trace  *app.InferTrace `json:"trace,omitempty"`
	Policy *app.PolicyInfo `json:"policy,omitempty"`
}
