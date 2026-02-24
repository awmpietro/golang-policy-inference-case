package httptransport

import (
	"encoding/json"
	"net/http"
)

type Inferer interface {
	Infer(policyDOT string, input map[string]any) (map[string]any, error)
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
}

type InferResponse struct {
	Output map[string]any `json:"output"`
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
