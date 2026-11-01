package formula

import (
	"fmt"
	"strings"
)

// Node is the common interface for all AST nodes.
type Node interface {
	Pos() int
	String() string
	Evaluate(ctx EvaluationContext) (Decimal, error)
	ExtractVariables(vars map[string]struct{})
	CountNodes() int
	MaxDepth() int
}

// NumberNode represents a literal numeric value.
type NumberNode struct {
	Value Decimal
	pos   int
}

func (n *NumberNode) Pos() int                                    { return n.pos }
func (n *NumberNode) String() string                              { return n.Value.String() }
func (n *NumberNode) Evaluate(ctx EvaluationContext) (Decimal, error) { return n.Value, nil }
func (n *NumberNode) ExtractVariables(vars map[string]struct{})   {}
func (n *NumberNode) CountNodes() int                             { return 1 }
func (n *NumberNode) MaxDepth() int                               { return 1 }

// VariableNode represents a named variable (e.g. area, maintenance_rate).
type VariableNode struct {
	Name string
	pos  int
}

func (v *VariableNode) Pos() int        { return v.pos }
func (v *VariableNode) String() string  { return v.Name }
func (v *VariableNode) CountNodes() int { return 1 }
func (v *VariableNode) MaxDepth() int   { return 1 }
func (v *VariableNode) ExtractVariables(vars map[string]struct{}) {
	vars[v.Name] = struct{}{}
}

func (v *VariableNode) Evaluate(ctx EvaluationContext) (Decimal, error) {
	val, exists := ctx.Variables[v.Name]
	if !exists {
		return Zero(), fmt.Errorf("missing variable: %q at position %d", v.Name, v.pos)
	}
	return val, nil
}

// UnaryNode represents unary operators (+a, -b).
type UnaryNode struct {
	Op   TokenType
	Expr Node
	pos  int
}

func (u *UnaryNode) Pos() int        { return u.pos }
func (u *UnaryNode) String() string  { return fmt.Sprintf("(%s%s)", u.Op.String(), u.Expr.String()) }
func (u *UnaryNode) CountNodes() int { return 1 + u.Expr.CountNodes() }
func (u *UnaryNode) MaxDepth() int   { return 1 + u.Expr.MaxDepth() }
func (u *UnaryNode) ExtractVariables(vars map[string]struct{}) {
	u.Expr.ExtractVariables(vars)
}

func (u *UnaryNode) Evaluate(ctx EvaluationContext) (Decimal, error) {
	val, err := u.Expr.Evaluate(ctx)
	if err != nil {
		return Zero(), err
	}
	switch u.Op {
	case TokenMinus:
		return val.Neg(), nil
	case TokenPlus:
		return val, nil
	default:
		return Zero(), fmt.Errorf("unsupported unary operator: %s at position %d", u.Op.String(), u.pos)
	}
}

// BinaryNode represents arithmetic operations (+, -, *, /).
type BinaryNode struct {
	Op    TokenType
	Left  Node
	Right Node
	pos   int
}

func (b *BinaryNode) Pos() int { return b.pos }
func (b *BinaryNode) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left.String(), b.Op.String(), b.Right.String())
}
func (b *BinaryNode) CountNodes() int { return 1 + b.Left.CountNodes() + b.Right.CountNodes() }
func (b *BinaryNode) MaxDepth() int {
	ld := b.Left.MaxDepth()
	rd := b.Right.MaxDepth()
	if ld > rd {
		return 1 + ld
	}
	return 1 + rd
}
func (b *BinaryNode) ExtractVariables(vars map[string]struct{}) {
	b.Left.ExtractVariables(vars)
	b.Right.ExtractVariables(vars)
}

func (b *BinaryNode) Evaluate(ctx EvaluationContext) (Decimal, error) {
	leftVal, err := b.Left.Evaluate(ctx)
	if err != nil {
		return Zero(), err
	}
	rightVal, err := b.Right.Evaluate(ctx)
	if err != nil {
		return Zero(), err
	}

	switch b.Op {
	case TokenPlus:
		return leftVal.Add(rightVal), nil
	case TokenMinus:
		return leftVal.Sub(rightVal), nil
	case TokenStar:
		return leftVal.Mul(rightVal), nil
	case TokenSlash:
		res, divErr := leftVal.Div(rightVal)
		if divErr != nil {
			return Zero(), fmt.Errorf("division by zero at position %d", b.pos)
		}
		return res, nil
	default:
		return Zero(), fmt.Errorf("unsupported binary operator: %s at position %d", b.Op.String(), b.pos)
	}
}

// ComparisonNode represents relational comparisons (>, >=, <, <=, ==, !=).
type ComparisonNode struct {
	Op    TokenType
	Left  Node
	Right Node
	pos   int
}

func (c *ComparisonNode) Pos() int { return c.pos }
func (c *ComparisonNode) String() string {
	return fmt.Sprintf("(%s %s %s)", c.Left.String(), c.Op.String(), c.Right.String())
}
func (c *ComparisonNode) CountNodes() int { return 1 + c.Left.CountNodes() + c.Right.CountNodes() }
func (c *ComparisonNode) MaxDepth() int {
	ld := c.Left.MaxDepth()
	rd := c.Right.MaxDepth()
	if ld > rd {
		return 1 + ld
	}
	return 1 + rd
}
func (c *ComparisonNode) ExtractVariables(vars map[string]struct{}) {
	c.Left.ExtractVariables(vars)
	c.Right.ExtractVariables(vars)
}

func (c *ComparisonNode) Evaluate(ctx EvaluationContext) (Decimal, error) {
	leftVal, err := c.Left.Evaluate(ctx)
	if err != nil {
		return Zero(), err
	}
	rightVal, err := c.Right.Evaluate(ctx)
	if err != nil {
		return Zero(), err
	}

	var res bool
	switch c.Op {
	case TokenGT:
		res = leftVal.GreaterThan(rightVal)
	case TokenGTE:
		res = leftVal.GreaterThanOrEqual(rightVal)
	case TokenLT:
		res = leftVal.LessThan(rightVal)
	case TokenLTE:
		res = leftVal.LessThanOrEqual(rightVal)
	case TokenEQ:
		res = leftVal.Equal(rightVal)
	case TokenNEQ:
		res = !leftVal.Equal(rightVal)
	default:
		return Zero(), fmt.Errorf("unsupported comparison operator: %s at position %d", c.Op.String(), c.pos)
	}

	if res {
		return One(), nil
	}
	return Zero(), nil
}

// ConditionalNode represents ternary if(cond, then, else) with short-circuit evaluation.
type ConditionalNode struct {
	Cond Node
	Then Node
	Else Node
	pos  int
}

func (c *ConditionalNode) Pos() int { return c.pos }
func (c *ConditionalNode) String() string {
	return fmt.Sprintf("if(%s, %s, %s)", c.Cond.String(), c.Then.String(), c.Else.String())
}
func (c *ConditionalNode) CountNodes() int {
	return 1 + c.Cond.CountNodes() + c.Then.CountNodes() + c.Else.CountNodes()
}
func (c *ConditionalNode) MaxDepth() int {
	cd := c.Cond.MaxDepth()
	td := c.Then.MaxDepth()
	ed := c.Else.MaxDepth()
	max := cd
	if td > max {
		max = td
	}
	if ed > max {
		max = ed
	}
	return 1 + max
}
func (c *ConditionalNode) ExtractVariables(vars map[string]struct{}) {
	c.Cond.ExtractVariables(vars)
	c.Then.ExtractVariables(vars)
	c.Else.ExtractVariables(vars)
}

func (c *ConditionalNode) Evaluate(ctx EvaluationContext) (Decimal, error) {
	condVal, err := c.Cond.Evaluate(ctx)
	if err != nil {
		return Zero(), err
	}

	// Short-circuit: if condition is non-zero (true), only evaluate Then
	if !condVal.IsZero() {
		return c.Then.Evaluate(ctx)
	}
	// Otherwise evaluate Else
	return c.Else.Evaluate(ctx)
}

// FunctionCallNode represents approved mathematical function invocations.
type FunctionCallNode struct {
	Name string
	Args []Node
	pos  int
}

func (f *FunctionCallNode) Pos() int { return f.pos }
func (f *FunctionCallNode) String() string {
	argStrs := make([]string, len(f.Args))
	for i, arg := range f.Args {
		argStrs[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", f.Name, strings.Join(argStrs, ", "))
}
func (f *FunctionCallNode) CountNodes() int {
	count := 1
	for _, arg := range f.Args {
		count += arg.CountNodes()
	}
	return count
}
func (f *FunctionCallNode) MaxDepth() int {
	max := 0
	for _, arg := range f.Args {
		d := arg.MaxDepth()
		if d > max {
			max = d
		}
	}
	return 1 + max
}
func (f *FunctionCallNode) ExtractVariables(vars map[string]struct{}) {
	for _, arg := range f.Args {
		arg.ExtractVariables(vars)
	}
}

func (f *FunctionCallNode) Evaluate(ctx EvaluationContext) (Decimal, error) {
	switch strings.ToLower(f.Name) {
	case "min":
		if len(f.Args) != 2 {
			return Zero(), fmt.Errorf("min() expects 2 arguments, got %d at position %d", len(f.Args), f.pos)
		}
		a, err := f.Args[0].Evaluate(ctx)
		if err != nil {
			return Zero(), err
		}
		b, err := f.Args[1].Evaluate(ctx)
		if err != nil {
			return Zero(), err
		}
		if a.LessThan(b) {
			return a, nil
		}
		return b, nil

	case "max":
		if len(f.Args) != 2 {
			return Zero(), fmt.Errorf("max() expects 2 arguments, got %d at position %d", len(f.Args), f.pos)
		}
		a, err := f.Args[0].Evaluate(ctx)
		if err != nil {
			return Zero(), err
		}
		b, err := f.Args[1].Evaluate(ctx)
		if err != nil {
			return Zero(), err
		}
		if a.GreaterThan(b) {
			return a, nil
		}
		return b, nil

	case "abs":
		if len(f.Args) != 1 {
			return Zero(), fmt.Errorf("abs() expects 1 argument, got %d at position %d", len(f.Args), f.pos)
		}
		val, err := f.Args[0].Evaluate(ctx)
		if err != nil {
			return Zero(), err
		}
		return val.Abs(), nil

	case "round":
		if len(f.Args) != 2 {
			return Zero(), fmt.Errorf("round() expects 2 arguments (value, precision), got %d at position %d", len(f.Args), f.pos)
		}
		val, err := f.Args[0].Evaluate(ctx)
		if err != nil {
			return Zero(), err
		}
		precVal, err := f.Args[1].Evaluate(ctx)
		if err != nil {
			return Zero(), err
		}
		prec := int(precVal.Float64())
		if prec < 0 || prec > 10 {
			return Zero(), fmt.Errorf("round() precision must be between 0 and 10, got %d at position %d", prec, f.pos)
		}
		return val.Round(prec), nil

	case "floor":
		if len(f.Args) != 1 {
			return Zero(), fmt.Errorf("floor() expects 1 argument, got %d at position %d", len(f.Args), f.pos)
		}
		val, err := f.Args[0].Evaluate(ctx)
		if err != nil {
			return Zero(), err
		}
		return val.Floor(), nil

	case "ceil":
		if len(f.Args) != 1 {
			return Zero(), fmt.Errorf("ceil() expects 1 argument, got %d at position %d", len(f.Args), f.pos)
		}
		val, err := f.Args[0].Evaluate(ctx)
		if err != nil {
			return Zero(), err
		}
		return val.Ceil(), nil

	default:
		return Zero(), fmt.Errorf("unknown function: %q at position %d", f.Name, f.pos)
	}
}
