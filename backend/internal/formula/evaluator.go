package formula

import (
	"fmt"
	"sort"
)

// EvaluationContext holds runtime variable bindings for formula calculation.
type EvaluationContext struct {
	Variables map[string]Decimal
}

// NewEvaluationContext initializes an empty EvaluationContext.
func NewEvaluationContext() EvaluationContext {
	return EvaluationContext{
		Variables: make(map[string]Decimal),
	}
}

// WithVariable sets or overrides a variable in context.
func (ctx EvaluationContext) WithVariable(name string, val Decimal) EvaluationContext {
	if ctx.Variables == nil {
		ctx.Variables = make(map[string]Decimal)
	}
	ctx.Variables[name] = val
	return ctx
}

// FormulaInfo contains structural metadata about a validated formula.
type FormulaInfo struct {
	Expression string   `json:"expression"`
	Variables  []string `json:"variables"`
	NodeCount  int      `json:"node_count"`
	MaxDepth   int      `json:"max_depth"`
}

// Evaluate parses and computes the result of a formula string against a given context.
func Evaluate(expr string, ctx EvaluationContext) (Decimal, error) {
	parser := NewParser(expr)
	ast, err := parser.Parse()
	if err != nil {
		return Zero(), err
	}
	return ast.Evaluate(ctx)
}

// Validate parses an expression and validates that all referenced variables are permitted.
func Validate(expr string, allowedVars []string) (*FormulaInfo, error) {
	parser := NewParser(expr)
	ast, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	varMap := make(map[string]struct{})
	ast.ExtractVariables(varMap)

	var list []string
	for v := range varMap {
		list = append(list, v)
	}
	sort.Strings(list)

	if len(allowedVars) > 0 {
		allowedMap := make(map[string]struct{}, len(allowedVars))
		for _, a := range allowedVars {
			allowedMap[a] = struct{}{}
		}
		for _, v := range list {
			if _, ok := allowedMap[v]; !ok {
				return nil, fmt.Errorf("unknown variable: %q is not in allowed list", v)
			}
		}
	}

	return &FormulaInfo{
		Expression: expr,
		Variables:  list,
		NodeCount:  ast.CountNodes(),
		MaxDepth:   ast.MaxDepth(),
	}, nil
}
