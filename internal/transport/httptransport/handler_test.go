package httptransport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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

type inferSvcDebugStub struct {
	inferSvcStub
	inferWithTraceFn func(policyDOT string, input map[string]any) (map[string]any, *app.InferTrace, error)
}

func (s *inferSvcDebugStub) InferWithTrace(policyDOT string, input map[string]any) (map[string]any, *app.InferTrace, error) {
	return s.inferWithTraceFn(policyDOT, input)
}

type inferSvcVersionedDebugStub struct {
	inferSvcVersionedStub
	inferWithTraceAndOptionsFn func(policyDOT string, input map[string]any, opts app.InferOptions) (map[string]any, *app.InferTrace, *app.PolicyInfo, error)
}

func (s *inferSvcVersionedDebugStub) InferWithTraceAndOptions(policyDOT string, input map[string]any, opts app.InferOptions) (map[string]any, *app.InferTrace, *app.PolicyInfo, error) {
	return s.inferWithTraceAndOptionsFn(policyDOT, input, opts)
}

func TestHandler_Infer_MethodNotAllowed(t *testing.T) {
	h := NewHandler(&inferSvcStub{inferFn: func(policyDOT string, input map[string]any) (map[string]any, error) {
		return map[string]any{}, nil
	}})

	req := httptest.NewRequest(http.MethodGet, "/infer", nil)
	rr := httptest.NewRecorder()

	h.Infer(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rr.Code)
	}
}

func TestHandler_Infer_InvalidJSON(t *testing.T) {
	h := NewHandler(&inferSvcStub{inferFn: func(policyDOT string, input map[string]any) (map[string]any, error) {
		return map[string]any{}, nil
	}})

	req := httptest.NewRequest(http.MethodPost, "/infer", bytes.NewBufferString("{"))
	rr := httptest.NewRecorder()

	h.Infer(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandler_Infer_DebugWithTrace(t *testing.T) {
	h := NewHandler(&inferSvcVersionedDebugStub{
		inferSvcVersionedStub: inferSvcVersionedStub{
			inferSvcStub: inferSvcStub{inferFn: func(policyDOT string, input map[string]any) (map[string]any, error) {
				return map[string]any{"approved": true}, nil
			}},
			inferWithOptionsFn: func(policyDOT string, input map[string]any, opts app.InferOptions) (map[string]any, *app.PolicyInfo, error) {
				return map[string]any{"approved": true}, &app.PolicyInfo{ID: opts.PolicyID, Version: opts.PolicyVersion, Hash: "h"}, nil
			},
		},
		inferWithTraceAndOptionsFn: func(policyDOT string, input map[string]any, opts app.InferOptions) (map[string]any, *app.InferTrace, *app.PolicyInfo, error) {
			return map[string]any{"approved": true}, &app.InferTrace{StartNode: "start", Terminated: "leaf"}, &app.PolicyInfo{ID: opts.PolicyID, Version: opts.PolicyVersion, Hash: "h"}, nil
		},
	})

	body := `{"policy_dot":"digraph{}","input":{"age":20},"debug":true,"policy_id":"credit","policy_version":"v1"}`
	req := httptest.NewRequest(http.MethodPost, "/infer", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.Infer(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["trace"] == nil {
		t.Fatalf("expected trace in response")
	}
	if out["policy"] == nil {
		t.Fatalf("expected policy metadata in response")
	}
}

func TestHandler_Infer_DebugFallbackWithoutTrace(t *testing.T) {
	h := NewHandler(&inferSvcStub{inferFn: func(policyDOT string, input map[string]any) (map[string]any, error) {
		return map[string]any{"approved": true}, nil
	}})

	body := `{"policy_dot":"digraph{}","input":{"age":20},"debug":true}`
	req := httptest.NewRequest(http.MethodPost, "/infer", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.Infer(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if _, ok := out["trace"]; ok {
		t.Fatalf("expected response without trace in fallback")
	}
}

func TestHandler_Infer_InvalidPolicyVersionPair(t *testing.T) {
	h := NewHandler(&inferSvcVersionedStub{
		inferSvcStub: inferSvcStub{inferFn: func(policyDOT string, input map[string]any) (map[string]any, error) {
			return map[string]any{"approved": true}, nil
		}},
		inferWithOptionsFn: func(policyDOT string, input map[string]any, opts app.InferOptions) (map[string]any, *app.PolicyInfo, error) {
			if (opts.PolicyID == "") != (opts.PolicyVersion == "") {
				return nil, &app.PolicyInfo{ID: opts.PolicyID, Version: opts.PolicyVersion, Hash: "h"}, fmt.Errorf("policy_id and policy_version must be provided together")
			}
			return map[string]any{"approved": true}, &app.PolicyInfo{ID: opts.PolicyID, Version: opts.PolicyVersion, Hash: "h"}, nil
		},
	})

	body := `{"policy_dot":"digraph{}","input":{"age":20},"policy_id":"credit"}`
	req := httptest.NewRequest(http.MethodPost, "/infer", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.Infer(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandler_Infer_VersionedResponseIncludesPolicyInfo(t *testing.T) {
	h := NewHandler(&inferSvcVersionedStub{
		inferSvcStub: inferSvcStub{inferFn: func(policyDOT string, input map[string]any) (map[string]any, error) {
			return map[string]any{"approved": true}, nil
		}},
		inferWithOptionsFn: func(policyDOT string, input map[string]any, opts app.InferOptions) (map[string]any, *app.PolicyInfo, error) {
			return map[string]any{"approved": true}, &app.PolicyInfo{ID: opts.PolicyID, Version: opts.PolicyVersion, Hash: "hash-1"}, nil
		},
	})

	body := `{"policy_dot":"digraph{}","input":{"age":20},"policy_id":"credit","policy_version":"v1"}`
	req := httptest.NewRequest(http.MethodPost, "/infer", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.Infer(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
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
