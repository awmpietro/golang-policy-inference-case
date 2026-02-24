package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/awmpietro/golang-policy-inference-case/internal/app"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy/cache"
	httptransport "github.com/awmpietro/golang-policy-inference-case/internal/transport/httptransport"
)

const policyThreePaths = `digraph Policy {
  start [result=""]
  approved [result="approved=true,segment=prime"]
  review [result="approved=false,segment=manual"]
  rejected [result="approved=false"]
  start -> approved [cond="age>=18 && score>700"]
  start -> review   [cond="age>=18 && score<=700"]
  start -> rejected [cond="age<18"]
}`

func newInferServer() *httptest.Server {
	compiler := policy.NewCompiler()
	engine := policy.NewEngine(policy.ExprEvaluator{})
	c := cache.NewInMemory(1024)
	svc := app.NewService(compiler, engine, c)
	h := httptransport.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/infer", h.Infer)
	return httptest.NewServer(mux)
}

func TestCompilerEngine_Integration(t *testing.T) {
	dotPath := filepath.Join("..", "policy", "testdata", "simple.dot")
	dot, err := os.ReadFile(dotPath)
	if err != nil {
		t.Fatal(err)
	}

	compiler := policy.NewCompiler()
	engine := policy.NewEngine(policy.ExprEvaluator{})

	compiledPolicy, err := compiler.Compile(string(dot))
	if err != nil {
		t.Fatal(err)
	}

	vars := map[string]any{"age": 20}
	if err := engine.Run(compiledPolicy, vars); err != nil {
		t.Fatal(err)
	}

	if vars["approved"] != true {
		t.Fatalf("expected approved=true, got %#v", vars["approved"])
	}
}

func postInfer(t *testing.T, srv *httptest.Server, rawBody string) (int, map[string]any, string) {
	t.Helper()

	resp, err := http.Post(srv.URL+"/infer", "application/json", bytes.NewBufferString(rawBody))
	if err != nil {
		t.Fatalf("post /infer failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response failed: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return resp.StatusCode, nil, string(body)
	}
	return resp.StatusCode, out, string(body)
}

func postInferJSON(t *testing.T, srv *httptest.Server, payload map[string]any) (int, map[string]any, string) {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}
	return postInfer(t, srv, string(b))
}

func TestHTTPInfer_EndToEndSuccess(t *testing.T) {
	srv := newInferServer()
	defer srv.Close()

	status, out, _ := postInferJSON(t, srv, map[string]any{
		"policy_dot": policyThreePaths,
		"input":      map[string]any{"age": 25, "score": 720},
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}

	output, ok := out["output"].(map[string]any)
	if !ok {
		t.Fatalf("missing output object: %#v", out)
	}
	if output["approved"] != true {
		t.Fatalf("expected approved=true, got %#v", output["approved"])
	}
	if output["segment"] != "prime" {
		t.Fatalf("expected segment=prime, got %#v", output["segment"])
	}
}

func TestHTTPInfer_CoversApprovedReviewRejectedPaths(t *testing.T) {
	srv := newInferServer()
	defer srv.Close()

	tests := []struct {
		name       string
		input      string
		wantStatus int
		wantSeg    string
		wantAppr   bool
	}{
		{name: "approved", input: `{"age":25,"score":720}`, wantStatus: 200, wantSeg: "prime", wantAppr: true},
		{name: "review", input: `{"age":25,"score":650}`, wantStatus: 200, wantSeg: "manual", wantAppr: false},
		{name: "rejected", input: `{"age":16,"score":900}`, wantStatus: 200, wantSeg: "", wantAppr: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var input map[string]any
			if err := json.Unmarshal([]byte(tc.input), &input); err != nil {
				t.Fatal(err)
			}
			status, out, _ := postInferJSON(t, srv, map[string]any{
				"policy_dot": policyThreePaths,
				"input":      input,
			})
			if status != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, status)
			}
			output := out["output"].(map[string]any)
			if output["approved"] != tc.wantAppr {
				t.Fatalf("expected approved=%v, got %#v", tc.wantAppr, output["approved"])
			}
			if tc.wantSeg == "" {
				if _, ok := output["segment"]; ok {
					t.Fatalf("expected no segment, got %#v", output["segment"])
				}
				return
			}
			if output["segment"] != tc.wantSeg {
				t.Fatalf("expected segment=%s, got %#v", tc.wantSeg, output["segment"])
			}
		})
	}
}

func TestHTTPInfer_InputErrors(t *testing.T) {
	srv := newInferServer()
	defer srv.Close()

	t.Run("invalid_json", func(t *testing.T) {
		status, _, _ := postInfer(t, srv, `{`)
		if status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", status)
		}
	})

	t.Run("invalid_policy_dot", func(t *testing.T) {
		status, out, _ := postInferJSON(t, srv, map[string]any{
			"policy_dot": "digraph { start -> ",
			"input":      map[string]any{"age": 20},
		})
		if status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", status)
		}
		if out["details"] == nil {
			t.Fatalf("expected error details")
		}
	})

	t.Run("missing_required_var_for_cond", func(t *testing.T) {
		status, out, _ := postInferJSON(t, srv, map[string]any{
			"policy_dot": `digraph { start -> approved [cond="age>=18 && score>700"]; approved [result="approved=true"]; }`,
			"input":      map[string]any{"age": 20},
		})
		if status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", status)
		}
		details, _ := out["details"].(string)
		if !strings.Contains(details, "missing input vars") {
			t.Fatalf("expected missing vars detail, got %q", details)
		}
	})
}

func TestHTTPInfer_DebugTrace(t *testing.T) {
	srv := newInferServer()
	defer srv.Close()

	status, out, _ := postInferJSON(t, srv, map[string]any{
		"policy_dot": policyThreePaths,
		"input":      map[string]any{"age": 25, "score": 720},
		"debug":      true,
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if out["trace"] == nil {
		t.Fatalf("expected trace in response")
	}
	trace := out["trace"].(map[string]any)
	if trace["terminated"] != "leaf" {
		t.Fatalf("expected terminated=leaf, got %#v", trace["terminated"])
	}
}

func TestHTTPInfer_PolicyVersioningValidAndInvalid(t *testing.T) {
	srv := newInferServer()
	defer srv.Close()

	t.Run("valid_pair", func(t *testing.T) {
		status, out, _ := postInferJSON(t, srv, map[string]any{
			"policy_dot":     policyThreePaths,
			"input":          map[string]any{"age": 25, "score": 720},
			"policy_id":      "credit",
			"policy_version": "v1",
		})
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}
		policyInfo := out["policy"].(map[string]any)
		if policyInfo["id"] != "credit" || policyInfo["version"] != "v1" {
			t.Fatalf("unexpected policy info: %#v", policyInfo)
		}
		if policyInfo["hash"] == "" {
			t.Fatalf("expected non-empty policy hash")
		}
	})

	t.Run("invalid_pair", func(t *testing.T) {
		status, out, _ := postInferJSON(t, srv, map[string]any{
			"policy_dot": policyThreePaths,
			"input":      map[string]any{"age": 25, "score": 720},
			"policy_id":  "credit",
		})
		if status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", status)
		}
		details, _ := out["details"].(string)
		if !strings.Contains(details, "policy_id and policy_version") {
			t.Fatalf("unexpected details: %q", details)
		}
	})
}

func TestHTTPInfer_RejectsCycleDOT(t *testing.T) {
	srv := newInferServer()
	defer srv.Close()

	status, out, _ := postInferJSON(t, srv, map[string]any{
		"policy_dot": `digraph { start -> a [cond="x==1"]; a -> b [cond="x==1"]; b -> a [cond="x==1"]; }`,
		"input":      map[string]any{"x": 1},
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}
	details, _ := out["details"].(string)
	if !strings.Contains(details, "contains cycle") {
		t.Fatalf("expected cycle error details, got %q", details)
	}
}

func TestHTTPInfer_ConcurrentRequests(t *testing.T) {
	srv := newInferServer()
	defer srv.Close()

	const n = 80
	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			age := 25
			score := 720
			if i%3 == 1 {
				score = 650
			}
			if i%3 == 2 {
				age = 16
			}
			status, out, body := postInferMapNoFatal(srv, map[string]any{
				"policy_dot": policyThreePaths,
				"input":      map[string]any{"age": age, "score": score},
			})
			if status != http.StatusOK {
				errs <- &integrationErr{msg: "status not ok", body: body}
				return
			}
			if out == nil || out["output"] == nil {
				errs <- &integrationErr{msg: "missing output", body: body}
				return
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

type integrationErr struct {
	msg  string
	body string
}

func (e *integrationErr) Error() string {
	return e.msg + ": " + e.body
}

func postInferMapNoFatal(srv *httptest.Server, payload map[string]any) (int, map[string]any, string) {
	b, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err.Error()
	}
	resp, err := http.Post(srv.URL+"/infer", "application/json", bytes.NewBuffer(b))
	if err != nil {
		return 0, nil, err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(body, &out)
	return resp.StatusCode, out, string(body)
}
