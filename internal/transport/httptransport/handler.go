package httptransport

import (
	"encoding/json"
	"net/http"

	"github.com/awmpietro/golang-policy-inference-case/internal/app"
	"github.com/awmpietro/golang-policy-inference-case/internal/transport/inferdto"
)

type Handler struct {
	svc app.InferService
}

func NewHandler(svc app.InferService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Infer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var in inferdto.InferRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json", "details": err.Error()})
		return
	}

	if in.Debug {
		out, trace, info, err := h.svc.InferWithTraceAndOptions(in.PolicyDOT, in.Input, in.Options())
		if err != nil {
			writeJSON(w, http.StatusBadRequest, inferErrorBody(err, trace, info))
			return
		}
		writeJSON(w, http.StatusOK, inferdto.InferResponse{Output: out, Trace: trace, Policy: info})
		return
	}

	out, info, err := h.svc.InferWithOptions(in.PolicyDOT, in.Input, in.Options())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, inferErrorBody(err, nil, info))
		return
	}
	writeJSON(w, http.StatusOK, inferdto.InferResponse{Output: out, Policy: info})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func inferErrorBody(err error, trace *app.InferTrace, info *app.PolicyInfo) map[string]any {
	body := map[string]any{
		"error":   "infer failed",
		"details": err.Error(),
	}
	if trace != nil {
		body["trace"] = trace
	}
	if info != nil {
		body["policy"] = info
	}
	return body
}
