package httptransport

import (
	"encoding/json"
	"net/http"

	"github.com/awmpietro/golang-policy-inference-case/internal/app"
)

type Inferer interface {
	Infer(policyDOT string, input map[string]any) (map[string]any, error)
}

type VersionedInferer interface {
	InferWithOptions(policyDOT string, input map[string]any, opts app.InferOptions) (map[string]any, *app.PolicyInfo, error)
}

type DebugInferer interface {
	InferWithTrace(policyDOT string, input map[string]any) (map[string]any, *app.InferTrace, error)
}

type VersionedDebugInferer interface {
	InferWithTraceAndOptions(policyDOT string, input map[string]any, opts app.InferOptions) (map[string]any, *app.InferTrace, *app.PolicyInfo, error)
}

type Handler struct {
	svc Inferer
}

func NewHandler(svc Inferer) *Handler {
	return &Handler{svc: svc}
}

type InferRequest struct {
	PolicyDOT string         `json:"policy_dot"`
	Input     map[string]any `json:"input"`
	PolicyID  string         `json:"policy_id,omitempty"`
	Version   string         `json:"policy_version,omitempty"`
	Debug     bool           `json:"debug,omitempty"`
}

type InferResponse struct {
	Output map[string]any  `json:"output"`
	Trace  *app.InferTrace `json:"trace,omitempty"`
	Policy *app.PolicyInfo `json:"policy,omitempty"`
}

func (h *Handler) Infer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var in InferRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json", "details": err.Error()})
		return
	}

	if in.Debug {
		opts := app.InferOptions{PolicyID: in.PolicyID, PolicyVersion: in.Version}
		if svc, ok := h.svc.(VersionedDebugInferer); ok {
			out, trace, info, err := svc.InferWithTraceAndOptions(in.PolicyDOT, in.Input, opts)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "infer failed", "details": err.Error(), "trace": trace, "policy": info})
				return
			}
			writeJSON(w, http.StatusOK, InferResponse{Output: out, Trace: trace, Policy: info})
			return
		}
		if svc, ok := h.svc.(DebugInferer); ok {
			out, trace, err := svc.InferWithTrace(in.PolicyDOT, in.Input)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "infer failed", "details": err.Error(), "trace": trace})
				return
			}
			writeJSON(w, http.StatusOK, InferResponse{Output: out, Trace: trace})
			return
		}
	}

	opts := app.InferOptions{PolicyID: in.PolicyID, PolicyVersion: in.Version}
	if svc, ok := h.svc.(VersionedInferer); ok {
		out, info, err := svc.InferWithOptions(in.PolicyDOT, in.Input, opts)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "infer failed", "details": err.Error(), "policy": info})
			return
		}
		writeJSON(w, http.StatusOK, InferResponse{Output: out, Policy: info})
		return
	}

	out, err := h.svc.Infer(in.PolicyDOT, in.Input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "infer failed", "details": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, InferResponse{Output: out})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
