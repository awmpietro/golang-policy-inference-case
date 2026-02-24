package api

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

func JSON(status int, body any) (events.APIGatewayV2HTTPResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		// fallback ultra simples
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       `{"error":"failed to encode response"}`,
			Headers:    map[string]string{"content-type": "application/json"},
		}, nil
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Body:       string(b),
		Headers:    map[string]string{"content-type": "application/json"},
	}, nil
}
