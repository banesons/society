package formula

import (
	"strings"
	"testing"
)

// 1. Lexer Tests
func TestLexer(t *testing.T) {
	input := `(area * 8.52) + if(overdue_days >= 30, min(interest, 500.00), 0) != -10`
	l := NewLexer(input)
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatalf("unexpected tokenize error: %v", err)
	}

	expectedTypes := []TokenType{
		TokenLParen, TokenIdent, TokenStar, TokenNumber, TokenRParen,
		TokenPlus,
		TokenIdent, TokenLParen, TokenIdent, TokenGTE, TokenNumber, TokenComma,
		TokenIdent, TokenLParen, TokenIdent, TokenComma, TokenNumber, TokenRParen, TokenComma,
		TokenNumber, TokenRParen,
		TokenNEQ, TokenMinus, TokenNumber,
		TokenEOF,
	}

	if len(tokens) != len(expectedTypes) {
		t.Fatalf("expected %d tokens, got %d", len(expectedTypes), len(tokens))
	}

	for i, exp := range expectedTypes {
		if tokens[i].Type != exp {
			t.Errorf("token[%d] expected type %v, got %v (%s)", i, exp, tokens[i].Type, tokens[i].Literal)
		}
	}
}

func TestLexer_InvalidChar(t *testing.T) {
	l := NewLexer("area @ 10")
	_, err := l.Tokenize()
	if err == nil {
		t.Fatal("expected error on invalid character '@', got nil")
	}
	if !strings.Contains(err.Error(), "unexpected character") {
		t.Errorf("expected error to mention unexpected character, got: %v", err)
	}
}

// 2. Parser & AST Tests
func TestParser_ValidExpressions(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		{
			name:     "simple binary",
			expr:     "area * rate",
			expected: "(area * rate)",
		},
		{
			name:     "precedence mult before add",
			expr:     "a + b * c",
			expected: "(a + (b * c))",
		},
		{
			name:     "parentheses override precedence",
			expr:     "(a + b) * c",
			expected: "((a + b) * c)",
		},
		{
			name:     "unary minus and plus",
			expr:     "-a + +b",
			expected: "((-a) + (+b))",
		},
		{
			name:     "comparisons",
			expr:     "area >= 1000 == true_val",
			expected: "((area >= 1000) == true_val)",
		},
		{
			name:     "conditional if",
			expr:     "if(area > 1000, area * rate, fixed_charge)",
			expected: "if((area > 1000), (area * rate), fixed_charge)",
		},
		{
			name:     "function calls",
			expr:     "min(a, max(b, c))",
			expected: "min(a, max(b, c))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.expr)
			ast, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if ast.String() != tt.expected {
				t.Errorf("expected AST string %q, got %q", tt.expected, ast.String())
			}
		})
	}
}

func TestParser_InvalidSyntax(t *testing.T) {
	invalidExprs := []struct {
		expr string
		desc string
	}{
		{"", "empty expression"},
		{"   ", "whitespace only"},
		{"a +", "dangling operator"},
		{"* a", "leading binary operator"},
		{"(a + b", "unclosed parenthesis"},
		{"a + b)", "unmatched closing parenthesis"},
		{"if(a > 0, b)", "if with insufficient arguments"},
		{"if(a > 0, b, c, d)", "if with too many arguments"},
		{"a b", "adjacent identifiers without operator"},
	}

	for _, tt := range invalidExprs {
		t.Run(tt.desc, func(t *testing.T) {
			p := NewParser(tt.expr)
			_, err := p.Parse()
			if err == nil {
				t.Errorf("expected parse error for %q (%s), got nil", tt.expr, tt.desc)
			}
		})
	}
}

// 3. Evaluator Tests
func TestEvaluator_Arithmetic(t *testing.T) {
	ctx := EvaluationContext{
		Variables: map[string]Decimal{
			"a": NewDecimalFromInt(10),
			"b": NewDecimalFromInt(20),
			"c": NewDecimalFromInt(3),
		},
	}

	tests := []struct {
		expr     string
		expected string
	}{
		{"a + b", "30"},
		{"b - a", "10"},
		{"a * b", "200"},
		{"b / a", "2"},
		{"a + b * c", "70"},
		{"(a + b) * c", "90"},
		{"-a + b", "10"},
		{"+b - a", "10"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			res, err := Evaluate(tt.expr, ctx)
			if err != nil {
				t.Fatalf("unexpected eval error: %v", err)
			}
			if res.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, res.String())
			}
		})
	}
}

func TestEvaluator_DivisionByZero(t *testing.T) {
	ctx := EvaluationContext{
		Variables: map[string]Decimal{
			"x": NewDecimalFromInt(10),
			"y": NewDecimalFromInt(0),
		},
	}

	_, err := Evaluate("x / y", ctx)
	if err == nil {
		t.Fatal("expected division by zero error, got nil")
	}
	if !strings.Contains(err.Error(), "division by zero") {
		t.Errorf("expected error to mention division by zero, got: %v", err)
	}
}

func TestEvaluator_MissingVariable(t *testing.T) {
	ctx := EvaluationContext{
		Variables: map[string]Decimal{
			"area": NewDecimalFromInt(865),
		},
	}

	_, err := Evaluate("area * maintenance_rate", ctx)
	if err == nil {
		t.Fatal("expected missing variable error, got nil")
	}
	if !strings.Contains(err.Error(), "missing variable") && !strings.Contains(err.Error(), "maintenance_rate") {
		t.Errorf("expected missing variable error mentioning maintenance_rate, got: %v", err)
	}
}

// 4. Comparisons and Conditionals (Short-Circuit)
func TestEvaluator_Conditionals(t *testing.T) {
	ctxLarge := EvaluationContext{
		Variables: map[string]Decimal{
			"area": NewDecimalFromInt(1200),
			"rate": NewDecimalFromFloat(8.5),
		},
	}
	resLarge, err := Evaluate("if(area > 1000, area * rate, 5000)", ctxLarge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resLarge.String() != "10200" {
		t.Errorf("expected 10200, got %s", resLarge.String())
	}

	ctxSmall := EvaluationContext{
		Variables: map[string]Decimal{
			"area": NewDecimalFromInt(800),
			"rate": NewDecimalFromFloat(8.5),
		},
	}
	resSmall, err := Evaluate("if(area > 1000, area * rate, 5000)", ctxSmall)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resSmall.String() != "5000" {
		t.Errorf("expected 5000, got %s", resSmall.String())
	}

	// Short-circuit proof: else branch has division by zero, but should NOT be executed when condition is true
	ctxZero := EvaluationContext{
		Variables: map[string]Decimal{
			"x": NewDecimalFromInt(1),
		},
	}
	resShortCircuit, err := Evaluate("if(x == 1, 100, 10 / 0)", ctxZero)
	if err != nil {
		t.Fatalf("short-circuit failed, got error: %v", err)
	}
	if resShortCircuit.String() != "100" {
		t.Errorf("expected 100, got %s", resShortCircuit.String())
	}
}

// 5. Approved Functions
func TestEvaluator_ApprovedFunctions(t *testing.T) {
	ctx := EvaluationContext{
		Variables: map[string]Decimal{
			"val1": NewDecimalFromFloat(15.75),
			"val2": NewDecimalFromFloat(22.50),
			"neg":  NewDecimalFromFloat(-42.80),
		},
	}

	tests := []struct {
		expr     string
		expected string
	}{
		{"min(val1, val2)", "15.75"},
		{"max(val1, val2)", "22.5"},
		{"abs(neg)", "42.8"},
		{"round(123.4567, 2)", "123.46"},
		{"round(123.4544, 2)", "123.45"},
		{"floor(15.75)", "15"},
		{"ceil(15.25)", "16"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			res, err := Evaluate(tt.expr, ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, res.String())
			}
		})
	}
}

// 6. Realistic Financial Billing Examples (Master Prompt Reference)
func TestFinancialCorrectness_MasterPromptReference(t *testing.T) {
	// Example 1: Unit 166 Base Maintenance
	// Area = 865 sq ft, Rate = 8.52 -> 7369.80
	ctx166 := EvaluationContext{
		Variables: map[string]Decimal{
			"area":                     NewDecimalFromInt(865),
			"maintenance_rate":         NewDecimalFromString("8.52"),
			"non_occupancy_percentage": NewDecimalFromInt(10),
			"water_consumption":        NewDecimalFromInt(173),
			"water_rate":               NewDecimalFromInt(10),
			"municipal_taxes":          NewDecimalFromString("3919.00"),
			"principal_arrears":        NewDecimalFromString("50000.00"),
			"interest_rate":            NewDecimalFromString("0.18"),
			"overdue_days":             NewDecimalFromInt(60),
			"advance_adjusted":         NewDecimalFromString("2000.00"),
		},
	}

	maintAmt, err := Evaluate("area * maintenance_rate", ctx166)
	if err != nil {
		t.Fatalf("failed to calculate maintenance: %v", err)
	}
	maintRounded := maintAmt.Round(2)
	if maintRounded.String() != "7369.8" && maintRounded.String() != "7369.80" {
		t.Errorf("expected maintenance amount 7369.80, got %s", maintRounded.String())
	}

	// Non-occupancy: maintenance_amount * 10 / 100 = 736.98
	ctx166.Variables["maintenance_amount"] = maintRounded
	nonOcc, err := Evaluate("maintenance_amount * non_occupancy_percentage / 100", ctx166)
	if err != nil {
		t.Fatalf("failed to calculate non occupancy: %v", err)
	}
	nonOccRounded := nonOcc.Round(2)
	if nonOccRounded.String() != "736.98" {
		t.Errorf("expected non-occupancy amount 736.98, got %s", nonOccRounded.String())
	}

	// Interest on arrears: principal_arrears * interest_rate * overdue_days / 365
	interest, err := Evaluate("principal_arrears * interest_rate * overdue_days / 365", ctx166)
	if err != nil {
		t.Fatalf("failed to calculate interest: %v", err)
	}
	interestRounded := interest.Round(2)
	if interestRounded.String() != "1479.45" {
		t.Errorf("expected interest 1479.45, got %s", interestRounded.String())
	}
}

// 7. Dependency Resolution & Cycle Detection
func TestDependencyResolver_TopologicalOrder(t *testing.T) {
	formulas := map[string]string{
		"subtotal":             "maintenance_amount + non_occupancy_amount",
		"non_occupancy_amount": "maintenance_amount * non_occupancy_rate",
		"maintenance_amount":   "area * maintenance_rate",
	}

	order, err := ResolveOrder(formulas)
	if err != nil {
		t.Fatalf("unexpected dependency error: %v", err)
	}

	// maintenance_amount must precede non_occupancy_amount, which must precede subtotal
	maintIdx, nonOccIdx, subtotalIdx := -1, -1, -1
	for i, name := range order {
		if name == "maintenance_amount" {
			maintIdx = i
		}
		if name == "non_occupancy_amount" {
			nonOccIdx = i
		}
		if name == "subtotal" {
			subtotalIdx = i
		}
	}

	if !(maintIdx < nonOccIdx && nonOccIdx < subtotalIdx) {
		t.Fatalf("invalid topological order: %v (maint: %d, nonOcc: %d, subtotal: %d)", order, maintIdx, nonOccIdx, subtotalIdx)
	}
}

func TestDependencyResolver_CircularDependency(t *testing.T) {
	// A -> B -> C -> A
	formulas := map[string]string{
		"charge_a": "charge_b + 10",
		"charge_b": "charge_c * 2",
		"charge_c": "charge_a - 5",
	}

	_, err := ResolveOrder(formulas)
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}
	if !strings.Contains(err.Error(), "circular dependency detected") {
		t.Errorf("expected error to mention circular dependency, got: %v", err)
	}
}

func TestDependencyResolver_SelfCycle(t *testing.T) {
	formulas := map[string]string{
		"charge_a": "charge_a + 1",
	}

	_, err := ResolveOrder(formulas)
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}
	if !strings.Contains(err.Error(), "circular dependency detected") {
		t.Errorf("expected error to mention circular dependency, got: %v", err)
	}
}

// 8. Batch Evaluation Pipeline
func TestEvaluateBatch(t *testing.T) {
	formulas := map[string]string{
		"maintenance_amount":   "area * rate",
		"non_occupancy_amount": "maintenance_amount * 0.10",
		"total_payable":        "maintenance_amount + non_occupancy_amount - advance",
	}

	ctx := EvaluationContext{
		Variables: map[string]Decimal{
			"area":    NewDecimalFromInt(1000),
			"rate":    NewDecimalFromInt(5),
			"advance": NewDecimalFromInt(500),
		},
	}

	results, err := EvaluateBatch(formulas, ctx)
	if err != nil {
		t.Fatalf("unexpected batch error: %v", err)
	}

	if results["maintenance_amount"].String() != "5000" {
		t.Errorf("expected maintenance_amount 5000, got %s", results["maintenance_amount"].String())
	}
	if results["non_occupancy_amount"].String() != "500" {
		t.Errorf("expected non_occupancy_amount 500, got %s", results["non_occupancy_amount"].String())
	}
	if results["total_payable"].String() != "5000" {
		t.Errorf("expected total_payable 5000, got %s", results["total_payable"].String())
	}
}

// 9. Security & Resource Limits
func TestSecurity_DisallowedFunctions(t *testing.T) {
	ctx := EvaluationContext{}
	_, err := Evaluate("eval(10 + 20)", ctx)
	if err == nil {
		t.Fatal("expected error for disallowed function 'eval', got nil")
	}
	if !strings.Contains(err.Error(), "unknown function") {
		t.Errorf("expected error to mention unknown function, got: %v", err)
	}

	_, err = Evaluate("system('ls')", ctx)
	if err == nil {
		t.Fatal("expected error for disallowed function 'system', got nil")
	}
}

func TestSecurity_ExpressionLimits(t *testing.T) {
	// Test max depth limit (> 30 parens)
	deepExpr := strings.Repeat("(", 35) + "10 + 20" + strings.Repeat(")", 35)
	p := NewParser(deepExpr)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected max depth error, got nil")
	}
	if !strings.Contains(err.Error(), "depth limit") {
		t.Errorf("expected depth limit error, got: %v", err)
	}
}

// 10. Benchmarks
func BenchmarkParse(b *testing.B) {
	expr := "if(area > 1000, (area * 8.52) + fixed_charge, min(water * 15, 500))"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewParser(expr)
		_, _ = p.Parse()
	}
}

func BenchmarkEvaluate(b *testing.B) {
	expr := "(area * 8.52) + if(overdue_days >= 30, min(interest, 500.00), 0)"
	p := NewParser(expr)
	ast, _ := p.Parse()

	ctx := EvaluationContext{
		Variables: map[string]Decimal{
			"area":         NewDecimalFromInt(865),
			"overdue_days": NewDecimalFromInt(45),
			"interest":     NewDecimalFromInt(650),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ast.Evaluate(ctx)
	}
}
