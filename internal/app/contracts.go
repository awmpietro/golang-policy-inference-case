package app

type InferService interface {
	InferWithOptions(policyDOT string, input map[string]any, opts InferOptions) (map[string]any, *PolicyInfo, error)
	InferWithTraceAndOptions(policyDOT string, input map[string]any, opts InferOptions) (map[string]any, *InferTrace, *PolicyInfo, error)
}
