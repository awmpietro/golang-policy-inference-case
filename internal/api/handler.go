package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

type InferRequest struct {
	PolicyDOT string         `json:"policy_dot"`
	Input     map[string]any `json:"input"`
}

type InferResponse struct {
	Output map[string]any `json:"output"`
}

func Handle(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	start := time.Now()
	defer func() {
		log.Printf("infer request processed in %s (bodyBytes=%d)", time.Since(start), len(req.Body))
	}()

	bodyBytes, err := readBody(req)
	if err != nil {
		return JSON(http.StatusBadRequest, map[string]any{
			"error":   "invalid body",
			"details": err.Error(),
		})
	}

	var in InferRequest
	if err := json.Unmarshal(bodyBytes, &in); err != nil {
		return JSON(http.StatusBadRequest, map[string]any{
			"error":   "invalid json",
			"details": err.Error(),
		})
	}

	if in.PolicyDOT == "" {
		return JSON(http.StatusBadRequest, map[string]any{
			"error": "policy_dot is required",
		})
	}
	if in.Input == nil {
		in.Input = map[string]any{}
	}

	return JSON(http.StatusOK, nil)
}

func readBody(req events.APIGatewayV2HTTPRequest) ([]byte, error) {
	if req.IsBase64Encoded {
		return base64.StdEncoding.DecodeString(req.Body)
	}
	return []byte(req.Body), nil
}
