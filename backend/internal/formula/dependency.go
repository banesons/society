package formula

import (
	"fmt"
	"sort"
	"strings"
)

// ResolveOrder analyzes dependencies between a set of named formulas
// and returns them in a deterministic topological order for sequential evaluation.
// Returns an error if a circular dependency is detected.
func ResolveOrder(formulas map[string]string) ([]string, error) {
	// Parse each formula and extract dependencies that are also keys in formulas
	deps := make(map[string][]string, len(formulas))
	names := make([]string, 0, len(formulas))

	for name, expr := range formulas {
		names = append(names, name)

		p := NewParser(expr)
		ast, err := p.Parse()
		if err != nil {
			return nil, fmt.Errorf("parse error in formula %q: %w", name, err)
		}

		varMap := make(map[string]struct{})
		ast.ExtractVariables(varMap)

		var localDeps []string
		for v := range varMap {
			if _, isFormula := formulas[v]; isFormula {
				localDeps = append(localDeps, v)
			}
		}
		sort.Strings(localDeps)
		deps[name] = localDeps
	}

	// Sort formula names alphabetically for deterministic tie-breaking
	sort.Strings(names)

	// Cycle detection and topological sort via DFS
	// Colors: 0 = unvisited, 1 = visiting (in recursion stack), 2 = visited
	visited := make(map[string]int, len(names))
	var result []string
	var recursionStack []string

	var visit func(node string) error
	visit = func(node string) error {
		visited[node] = 1 // visiting
		recursionStack = append(recursionStack, node)

		for _, dep := range deps[node] {
			if visited[dep] == 1 {
				// Cycle found! Extract cycle path
				cycleStart := -1
				for i, n := range recursionStack {
					if n == dep {
						cycleStart = i
						break
					}
				}
				var cyclePath []string
				if cycleStart != -1 {
					cyclePath = append(cyclePath, recursionStack[cycleStart:]...)
				} else {
					cyclePath = append(cyclePath, dep)
				}
				cyclePath = append(cyclePath, dep)
				return fmt.Errorf("circular dependency detected: %s", strings.Join(cyclePath, " -> "))
			}
			if visited[dep] == 0 {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}

		visited[node] = 2 // visited
		recursionStack = recursionStack[:len(recursionStack)-1]
		result = append(result, node)
		return nil
	}

	for _, name := range names {
		if visited[name] == 0 {
			if err := visit(name); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// EvaluateBatch resolves formulas in topological order and computes the full set of results.
func EvaluateBatch(formulas map[string]string, initialContext EvaluationContext) (map[string]Decimal, error) {
	order, err := ResolveOrder(formulas)
	if err != nil {
		return nil, err
	}

	// Copy initial context
	evalCtx := NewEvaluationContext()
	for k, v := range initialContext.Variables {
		evalCtx.Variables[k] = v
	}

	results := make(map[string]Decimal, len(formulas))
	for _, name := range order {
		expr := formulas[name]
		val, err := Evaluate(expr, evalCtx)
		if err != nil {
			return nil, fmt.Errorf("error evaluating formula %q: %w", name, err)
		}
		evalCtx.Variables[name] = val
		results[name] = val
	}

	return results, nil
}
