package lambdatransport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"github.com/awmpietro/golang-policy-inference-case/internal/app"
)

type Handler struct {
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

type InferRequest struct {
	PolicyDOT string         `json:"policy_dot"`
	Input     map[string]any `json:"input"`
}

type InferResponse struct {
	Output map[string]any `json:"output"`
}

// Infer assume que o API Gateway já roteou POST /infer.
func (h *Handler) Infer(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	body, err := readBody(req)
	if err != nil {
		return jsonResp(http.StatusBadRequest, map[string]any{"error": "invalid body", "details": err.Error()}), nil
	}

	var in InferRequest
	if err := json.Unmarshal(body, &in); err != nil {
		return jsonResp(http.StatusBadRequest, map[string]any{"error": "invalid json", "details": err.Error()}), nil
	}

	out, err := h.svc.Infer(in.PolicyDOT, in.Input)
	if err != nil {
		return jsonResp(http.StatusBadRequest, map[string]any{"error": "infer failed", "details": err.Error()}), nil
	}

	return jsonResp(http.StatusOK, InferResponse{Output: out}), nil
}

func readBody(req events.APIGatewayV2HTTPRequest) ([]byte, error) {
	if req.IsBase64Encoded {
		return base64.StdEncoding.DecodeString(req.Body)
	}
	return []byte(req.Body), nil
}

func jsonResp(status int, body any) events.APIGatewayV2HTTPResponse {
	b, _ := json.Marshal(body)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       string(b),
	}
}
