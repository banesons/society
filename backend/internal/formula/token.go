package formula

import (
	"fmt"
	"unicode"
)

// TokenType represents lexical token categories.
type TokenType int

const (
	TokenIllegal TokenType = iota
	TokenEOF
	TokenNumber
	TokenIdent
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenLParen
	TokenRParen
	TokenComma
	TokenGT
	TokenGTE
	TokenLT
	TokenLTE
	TokenEQ
	TokenNEQ
)

var tokenNames = map[TokenType]string{
	TokenIllegal: "ILLEGAL",
	TokenEOF:     "EOF",
	TokenNumber:  "NUMBER",
	TokenIdent:   "IDENT",
	TokenPlus:    "+",
	TokenMinus:   "-",
	TokenStar:    "*",
	TokenSlash:   "/",
	TokenLParen:  "(",
	TokenRParen:  ")",
	TokenComma:   ",",
	TokenGT:      ">",
	TokenGTE:     ">=",
	TokenLT:      "<",
	TokenLTE:     "<=",
	TokenEQ:      "==",
	TokenNEQ:     "!=",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TOKEN(%d)", t)
}

// Token represents a lexical token with its position in source text.
type Token struct {
	Type    TokenType
	Literal string
	Pos     int
}

// Lexer breaks a formula string into a sequence of tokens.
type Lexer struct {
	input   string
	pos     int
	readPos int
	ch      byte
}

// NewLexer creates a Lexer for the input expression.
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// Tokenize reads all tokens until EOF or returns a lexical error.
func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token
	for {
		tok, err := l.NextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens, nil
}

// NextToken scans and returns the next token.
func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()

	startPos := l.pos
	var tok Token

	switch l.ch {
	case '+':
		tok = Token{Type: TokenPlus, Literal: "+", Pos: startPos}
		l.readChar()
	case '-':
		tok = Token{Type: TokenMinus, Literal: "-", Pos: startPos}
		l.readChar()
	case '*':
		tok = Token{Type: TokenStar, Literal: "*", Pos: startPos}
		l.readChar()
	case '/':
		tok = Token{Type: TokenSlash, Literal: "/", Pos: startPos}
		l.readChar()
	case '(':
		tok = Token{Type: TokenLParen, Literal: "(", Pos: startPos}
		l.readChar()
	case ')':
		tok = Token{Type: TokenRParen, Literal: ")", Pos: startPos}
		l.readChar()
	case ',':
		tok = Token{Type: TokenComma, Literal: ",", Pos: startPos}
		l.readChar()
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TokenGTE, Literal: ">=", Pos: startPos}
		} else {
			tok = Token{Type: TokenGT, Literal: ">", Pos: startPos}
		}
		l.readChar()
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TokenLTE, Literal: "<=", Pos: startPos}
		} else {
			tok = Token{Type: TokenLT, Literal: "<", Pos: startPos}
		}
		l.readChar()
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TokenEQ, Literal: "==", Pos: startPos}
			l.readChar()
		} else {
			return Token{}, fmt.Errorf("unexpected character '=' at position %d, did you mean '=='?", startPos)
		}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TokenNEQ, Literal: "!=", Pos: startPos}
			l.readChar()
		} else {
			return Token{}, fmt.Errorf("unexpected character '!' at position %d, did you mean '!='?", startPos)
		}
	case 0:
		tok = Token{Type: TokenEOF, Literal: "", Pos: startPos}
	default:
		if isDigit(l.ch) {
			numLit := l.readNumber()
			return Token{Type: TokenNumber, Literal: numLit, Pos: startPos}, nil
		} else if isLetter(l.ch) {
			ident := l.readIdentifier()
			return Token{Type: TokenIdent, Literal: ident, Pos: startPos}, nil
		} else {
			return Token{}, fmt.Errorf("unexpected character '%c' at position %d", l.ch, startPos)
		}
	}

	return tok, nil
}

func (l *Lexer) readNumber() string {
	start := l.pos
	hasDot := false
	for isDigit(l.ch) || (l.ch == '.' && !hasDot) {
		if l.ch == '.' {
			hasDot = true
		}
		l.readChar()
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readIdentifier() string {
	start := l.pos
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}
