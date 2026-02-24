package lambdatransport

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"github.com/awmpietro/golang-policy-inference-case/internal/app"
)

type inferSvcStub struct {
	inferFn func(policyDOT string, input map[string]any) (map[string]any, error)
}

func (s *inferSvcStub) Infer(policyDOT string, input map[string]any) (map[string]any, error) {
	return s.inferFn(policyDOT, input)
}

type inferSvcVersionedStub struct {
	inferSvcStub
	inferWithOptionsFn func(policyDOT string, input map[string]any, opts app.InferOptions) (map[string]any, *app.PolicyInfo, error)
}

func (s *inferSvcVersionedStub) InferWithOptions(policyDOT string, input map[string]any, opts app.InferOptions) (map[string]any, *app.PolicyInfo, error) {
	return s.inferWithOptionsFn(policyDOT, input, opts)
}

func TestHandler_Infer_InvalidJSON(t *testing.T) {
	h := NewHandler(&inferSvcStub{inferFn: func(policyDOT string, input map[string]any) (map[string]any, error) {
		return map[string]any{"approved": true}, nil
	}})

	resp, err := h.Infer(context.Background(), events.APIGatewayV2HTTPRequest{Body: "{"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandler_Infer_VersionedResponseIncludesPolicyInfo(t *testing.T) {
	h := NewHandler(&inferSvcVersionedStub{
		inferSvcStub: inferSvcStub{inferFn: func(policyDOT string, input map[string]any) (map[string]any, error) {
			return map[string]any{"approved": true}, nil
		}},
		inferWithOptionsFn: func(policyDOT string, input map[string]any, opts app.InferOptions) (map[string]any, *app.PolicyInfo, error) {
			if (opts.PolicyID == "") != (opts.PolicyVersion == "") {
				return nil, nil, fmt.Errorf("policy_id and policy_version must be provided together")
			}
			return map[string]any{"approved": true}, &app.PolicyInfo{ID: opts.PolicyID, Version: opts.PolicyVersion, Hash: "hash-1"}, nil
		},
	})

	body := `{"policy_dot":"digraph{}","input":{"age":20},"policy_id":"credit","policy_version":"v1"}`
	resp, err := h.Infer(context.Background(), events.APIGatewayV2HTTPRequest{Body: body})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(resp.Body), &out); err != nil {
		t.Fatal(err)
	}
	policy, ok := out["policy"].(map[string]any)
	if !ok {
		t.Fatalf("expected policy object in response, got %#v", out["policy"])
	}
	if policy["id"] != "credit" || policy["version"] != "v1" || policy["hash"] != "hash-1" {
		t.Fatalf("unexpected policy info: %#v", policy)
	}
}
