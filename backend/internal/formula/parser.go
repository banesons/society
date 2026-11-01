package formula

import (
	"fmt"
	"strings"
)

const (
	MaxASTDepth = 30
	MaxASTNodes = 200
)

// Parser parses a token stream into an Abstract Syntax Tree (AST).
type Parser struct {
	tokens []Token
	pos    int
	depth  int
}

// NewParser creates a new Parser for an expression string.
func NewParser(expr string) *Parser {
	l := NewLexer(expr)
	tokens, _ := l.Tokenize()
	return &Parser{tokens: tokens, pos: 0, depth: 0}
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) match(types ...TokenType) bool {
	cur := p.current().Type
	for _, t := range types {
		if cur == t {
			p.advance()
			return true
		}
	}
	return false
}

// Parse parses the entire expression and validates resource limits.
func (p *Parser) Parse() (Node, error) {
	if len(p.tokens) == 0 || p.current().Type == TokenEOF {
		return nil, fmt.Errorf("empty expression")
	}

	node, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token %q at position %d", p.current().Literal, p.current().Pos)
	}

	if node.MaxDepth() > MaxASTDepth {
		return nil, fmt.Errorf("expression depth limit exceeded: depth %d (max %d)", node.MaxDepth(), MaxASTDepth)
	}

	if node.CountNodes() > MaxASTNodes {
		return nil, fmt.Errorf("expression complexity limit exceeded: %d nodes (max %d)", node.CountNodes(), MaxASTNodes)
	}

	return node, nil
}

// parseExpression = parseComparison
func (p *Parser) parseExpression() (Node, error) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > MaxASTDepth {
		return nil, fmt.Errorf("expression depth limit exceeded: depth %d (max %d)", p.depth, MaxASTDepth)
	}

	return p.parseComparison()
}

// parseComparison = parseAdditive ( ('>' | '>=' | '<' | '<=' | '==' | '!=') parseAdditive )*
func (p *Parser) parseComparison() (Node, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}

	for {
		cur := p.current()
		if cur.Type == TokenGT || cur.Type == TokenGTE || cur.Type == TokenLT ||
			cur.Type == TokenLTE || cur.Type == TokenEQ || cur.Type == TokenNEQ {
			opTok := p.advance()
			right, err := p.parseAdditive()
			if err != nil {
				return nil, err
			}
			left = &ComparisonNode{
				Op:    opTok.Type,
				Left:  left,
				Right: right,
				pos:   opTok.Pos,
			}
		} else {
			break
		}
	}

	return left, nil
}

// parseAdditive = parseMultiplicative ( ('+' | '-') parseMultiplicative )*
func (p *Parser) parseAdditive() (Node, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}

	for {
		cur := p.current()
		if cur.Type == TokenPlus || cur.Type == TokenMinus {
			opTok := p.advance()
			right, err := p.parseMultiplicative()
			if err != nil {
				return nil, err
			}
			left = &BinaryNode{
				Op:    opTok.Type,
				Left:  left,
				Right: right,
				pos:   opTok.Pos,
			}
		} else {
			break
		}
	}

	return left, nil
}

// parseMultiplicative = parseUnary ( ('*' | '/') parseUnary )*
func (p *Parser) parseMultiplicative() (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		cur := p.current()
		if cur.Type == TokenStar || cur.Type == TokenSlash {
			opTok := p.advance()
			right, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			left = &BinaryNode{
				Op:    opTok.Type,
				Left:  left,
				Right: right,
				pos:   opTok.Pos,
			}
		} else {
			break
		}
	}

	return left, nil
}

// parseUnary = ('+' | '-') parseUnary | parsePrimary
func (p *Parser) parseUnary() (Node, error) {
	cur := p.current()
	if cur.Type == TokenPlus || cur.Type == TokenMinus {
		opTok := p.advance()
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{
			Op:   opTok.Type,
			Expr: expr,
			pos:  opTok.Pos,
		}, nil
	}

	return p.parsePrimary()
}

// parsePrimary = NUMBER | '(' expr ')' | if(...) | func(...) | IDENT
func (p *Parser) parsePrimary() (Node, error) {
	cur := p.current()

	switch cur.Type {
	case TokenNumber:
		tok := p.advance()
		dec := NewDecimalFromString(tok.Literal)
		return &NumberNode{Value: dec, pos: tok.Pos}, nil

	case TokenLParen:
		lParen := p.advance()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.current().Type != TokenRParen {
			return nil, fmt.Errorf("unclosed parenthesis starting at position %d", lParen.Pos)
		}
		p.advance() // consume ')'
		return expr, nil

	case TokenIdent:
		tok := p.advance()

		// Check for conditional if(...)
		if strings.ToLower(tok.Literal) == "if" && p.current().Type == TokenLParen {
			return p.parseConditional(tok.Pos)
		}

		// Check for function call ident(...)
		if p.current().Type == TokenLParen {
			return p.parseFunctionCall(tok.Literal, tok.Pos)
		}

		return &VariableNode{Name: tok.Literal, pos: tok.Pos}, nil

	default:
		return nil, fmt.Errorf("unexpected token %q at position %d", cur.Literal, cur.Pos)
	}
}

func (p *Parser) parseConditional(startPos int) (Node, error) {
	p.advance() // consume '('

	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenComma {
		return nil, fmt.Errorf("expected ',' after condition in if() at position %d", p.current().Pos)
	}
	p.advance() // consume ','

	thenNode, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenComma {
		return nil, fmt.Errorf("expected ',' after then branch in if() at position %d", p.current().Pos)
	}
	p.advance() // consume ','

	elseNode, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' to close if() at position %d", p.current().Pos)
	}
	p.advance() // consume ')'

	return &ConditionalNode{
		Cond: cond,
		Then: thenNode,
		Else: elseNode,
		pos:  startPos,
	}, nil
}

func (p *Parser) parseFunctionCall(name string, startPos int) (Node, error) {
	p.advance() // consume '('

	var args []Node
	if p.current().Type != TokenRParen {
		for {
			arg, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if p.current().Type == TokenComma {
				p.advance() // consume ','
			} else {
				break
			}
		}
	}

	if p.current().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' closing function %s() at position %d", name, p.current().Pos)
	}
	p.advance() // consume ')'

	return &FunctionCallNode{
		Name: name,
		Args: args,
		pos:  startPos,
	}, nil
}
