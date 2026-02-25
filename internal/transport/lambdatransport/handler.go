package lambdatransport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"github.com/awmpietro/golang-policy-inference-case/internal/app"
	"github.com/awmpietro/golang-policy-inference-case/internal/transport/inferdto"
)

type Handler struct {
	svc app.InferService
}

func NewHandler(svc app.InferService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Infer(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	body, err := readBody(req)
	if err != nil {
		return jsonResp(http.StatusBadRequest, map[string]any{"error": "invalid body", "details": err.Error()}), nil
	}

	var in inferdto.InferRequest
	if err := json.Unmarshal(body, &in); err != nil {
		return jsonResp(http.StatusBadRequest, map[string]any{"error": "invalid json", "details": err.Error()}), nil
	}

	if in.Debug {
		out, trace, info, err := h.svc.InferWithTraceAndOptions(in.PolicyDOT, in.Input, in.Options())
		if err != nil {
			return jsonResp(http.StatusBadRequest, inferErrorBody(err, trace, info)), nil
		}
		return jsonResp(http.StatusOK, inferdto.InferResponse{Output: out, Trace: trace, Policy: info}), nil
	}

	out, info, err := h.svc.InferWithOptions(in.PolicyDOT, in.Input, in.Options())
	if err != nil {
		return jsonResp(http.StatusBadRequest, inferErrorBody(err, nil, info)), nil
	}
	return jsonResp(http.StatusOK, inferdto.InferResponse{Output: out, Policy: info}), nil
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
