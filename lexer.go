package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type TokenType int

const (
	// Literals
	TOKEN_NUMBER TokenType = iota
	TOKEN_STRING
	TOKEN_TEMPLATE_STRING
	TOKEN_IDENT
	TOKEN_TRUE
	TOKEN_FALSE
	TOKEN_NIL

	// Keywords
	TOKEN_AND
	TOKEN_BREAK
	TOKEN_CONTINUE
	TOKEN_CASE
	TOKEN_DEFAULT
	TOKEN_DO
	TOKEN_ELSE
	TOKEN_ELSEIF
	TOKEN_END
	TOKEN_FOR
	TOKEN_FUNCTION
	TOKEN_GLOBAL
	TOKEN_GOTO
	TOKEN_IF
	TOKEN_IN
	TOKEN_LOCAL
	TOKEN_NOT
	TOKEN_OR
	TOKEN_REPEAT
	TOKEN_RETURN
	TOKEN_SWITCH
	TOKEN_THEN
	TOKEN_UNTIL
	TOKEN_WHILE

	// Operators
	TOKEN_PLUS            // +
	TOKEN_MINUS           // -
	TOKEN_STAR            // *
	TOKEN_SLASH           // /
	TOKEN_PERCENT         // %
	TOKEN_CARET           // ^
	TOKEN_HASH            // #
	TOKEN_EQ              // ==
	TOKEN_NEQ             // ~= or !=
	TOKEN_LT              // <
	TOKEN_GT              // >
	TOKEN_LE              // <=
	TOKEN_GE              // >=
	TOKEN_ASSIGN          // =
	TOKEN_LPAREN          // (
	TOKEN_RPAREN          // )
	TOKEN_LBRACE          // {
	TOKEN_RBRACE          // }
	TOKEN_LBRACKET        // [
	TOKEN_RBRACKET        // ]
	TOKEN_SEMICOLON       // ;
	TOKEN_COLON           // :
	TOKEN_DOUBLECOLON     // ::
	TOKEN_COMMA           // ,
	TOKEN_DOT             // .
	TOKEN_DOTDOT          // ..
	TOKEN_DOTDOTDOT       // ...
	TOKEN_QUESTION_DOT    // ?.
	TOKEN_DOUBLE_QUESTION // ??

	// Compound assignment
	TOKEN_PLUS_ASSIGN    // +=
	TOKEN_MINUS_ASSIGN   // -=
	TOKEN_STAR_ASSIGN    // *=
	TOKEN_SLASH_ASSIGN   // /=
	TOKEN_PERCENT_ASSIGN // %=
	TOKEN_DOTDOT_ASSIGN  // ..=

	TOKEN_NEWLINE
	TOKEN_EOF
	TOKEN_ERROR
)

var keywords = map[string]TokenType{
	"and":      TOKEN_AND,
	"break":    TOKEN_BREAK,
	"continue": TOKEN_CONTINUE,
	"case":     TOKEN_CASE,
	"default":  TOKEN_DEFAULT,
	"do":       TOKEN_DO,
	"else":     TOKEN_ELSE,
	"elseif":   TOKEN_ELSEIF,
	"end":      TOKEN_END,
	"false":    TOKEN_FALSE,
	"for":      TOKEN_FOR,
	"function": TOKEN_FUNCTION,
	"global":   TOKEN_GLOBAL,
	"goto":     TOKEN_GOTO,
	"if":       TOKEN_IF,
	"in":       TOKEN_IN,
	"local":    TOKEN_LOCAL,
	"nil":      TOKEN_NIL,
	"not":      TOKEN_NOT,
	"or":       TOKEN_OR,
	"repeat":   TOKEN_REPEAT,
	"return":   TOKEN_RETURN,
	"switch":   TOKEN_SWITCH,
	"then":     TOKEN_THEN,
	"true":     TOKEN_TRUE,
	"until":    TOKEN_UNTIL,
	"while":    TOKEN_WHILE,
}

type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

func (t Token) String() string {
	return fmt.Sprintf("Token{%v, %q, line %d}", t.Type, t.Value, t.Line)
}

type Lexer struct {
	source      string
	tokens      []Token
	start       int
	current     int
	line        int
	column      int
	startColumn int
}

func NewLexer(source string) *Lexer {
	return &Lexer{
		source:  source,
		tokens:  []Token{},
		start:   0,
		current: 0,
		line:    1,
		column:  1,
	}
}

func (l *Lexer) Tokenize() ([]Token, error) {
	for !l.isAtEnd() {
		l.start = l.current
		l.startColumn = l.column
		if err := l.scanToken(); err != nil {
			return nil, err
		}
	}
	l.tokens = append(l.tokens, Token{Type: TOKEN_EOF, Value: "", Line: l.line, Column: l.column})
	return l.tokens, nil
}

func (l *Lexer) scanToken() error {
	c := l.advance()

	switch c {
	case '(':
		l.addToken(TOKEN_LPAREN)
	case ')':
		l.addToken(TOKEN_RPAREN)
	case '{':
		l.addToken(TOKEN_LBRACE)
	case '}':
		l.addToken(TOKEN_RBRACE)
	case '[':
		// Check for multiline string [[...]]
		if l.peek() == '[' || l.peek() == '=' {
			return l.multilineString()
		}
		l.addToken(TOKEN_LBRACKET)
	case ']':
		l.addToken(TOKEN_RBRACKET)
	case ';':
		l.addToken(TOKEN_SEMICOLON)
	case ',':
		l.addToken(TOKEN_COMMA)
	case '#':
		l.addToken(TOKEN_HASH)
	case '^':
		l.addToken(TOKEN_CARET)

	case '+':
		if l.match('=') {
			l.addToken(TOKEN_PLUS_ASSIGN)
		} else {
			l.addToken(TOKEN_PLUS)
		}
	case '-':
		if l.match('-') {
			l.comment()
		} else if l.match('=') {
			l.addToken(TOKEN_MINUS_ASSIGN)
		} else {
			l.addToken(TOKEN_MINUS)
		}
	case '*':
		if l.match('=') {
			l.addToken(TOKEN_STAR_ASSIGN)
		} else {
			l.addToken(TOKEN_STAR)
		}
	case '/':
		if l.match('=') {
			l.addToken(TOKEN_SLASH_ASSIGN)
		} else {
			l.addToken(TOKEN_SLASH)
		}
	case '%':
		if l.match('=') {
			l.addToken(TOKEN_PERCENT_ASSIGN)
		} else {
			l.addToken(TOKEN_PERCENT)
		}

	case '=':
		if l.match('=') {
			l.addToken(TOKEN_EQ)
		} else {
			l.addToken(TOKEN_ASSIGN)
		}
	case '!':
		if l.match('=') {
			l.addToken(TOKEN_NEQ)
		} else {
			return fmt.Errorf("line %d: unexpected character '!'", l.line)
		}
	case '~':
		if l.match('=') {
			l.addToken(TOKEN_NEQ)
		} else {
			return fmt.Errorf("line %d: unexpected character '~' (did you mean '~='?)", l.line)
		}
	case '<':
		if l.match('=') {
			l.addToken(TOKEN_LE)
		} else {
			l.addToken(TOKEN_LT)
		}
	case '>':
		if l.match('=') {
			l.addToken(TOKEN_GE)
		} else {
			l.addToken(TOKEN_GT)
		}

	case ':':
		if l.match(':') {
			l.addToken(TOKEN_DOUBLECOLON)
		} else {
			l.addToken(TOKEN_COLON)
		}
	case '.':
		if l.match('.') {
			if l.match('.') {
				l.addToken(TOKEN_DOTDOTDOT)
			} else if l.match('=') {
				l.addToken(TOKEN_DOTDOT_ASSIGN)
			} else {
				l.addToken(TOKEN_DOTDOT)
			}
		} else if isDigit(l.peek()) {
			l.number()
		} else {
			l.addToken(TOKEN_DOT)
		}
	case '?':
		if l.match('.') {
			l.addToken(TOKEN_QUESTION_DOT)
		} else if l.match('?') {
			l.addToken(TOKEN_DOUBLE_QUESTION)
		} else {
			return fmt.Errorf("line %d: unexpected character '?'", l.line)
		}

	case '"', '\'':
		return l.string(c)
	case '`':
		return l.templateString()

	case '\n':
		l.line++
		l.column = 1
	case ' ', '\r', '\t':
		// Ignore whitespace
	default:
		if isDigit(c) {
			l.number()
		} else if isAlpha(c) {
			l.identifier()
		} else {
			return fmt.Errorf("line %d: unexpected character '%c'", l.line, c)
		}
	}
	return nil
}

func (l *Lexer) comment() {
	// Check for multiline comment --[[...]]
	if l.peek() == '[' && (l.peekNext() == '[' || l.peekNext() == '=') {
		l.advance() // consume '['
		// Count equals signs
		eqCount := 0
		for l.peek() == '=' {
			l.advance()
			eqCount++
		}
		if l.peek() != '[' {
			// Not a valid multiline comment start, treat as single line
			for l.peek() != '\n' && !l.isAtEnd() {
				l.advance()
			}
			return
		}
		l.advance() // consume second '['

		// Find matching ]=*]
		for !l.isAtEnd() {
			if l.peek() == '\n' {
				l.line++
				l.column = 0
			}
			if l.peek() == ']' {
				l.advance()
				matchEq := 0
				for l.peek() == '=' && matchEq < eqCount {
					l.advance()
					matchEq++
				}
				if matchEq == eqCount && l.peek() == ']' {
					l.advance()
					return
				}
			} else {
				l.advance()
			}
		}
	} else {
		// Single line comment
		for l.peek() != '\n' && !l.isAtEnd() {
			l.advance()
		}
	}
}

func (l *Lexer) string(quote byte) error {
	for l.peek() != quote && !l.isAtEnd() {
		if l.peek() == '\n' {
			return fmt.Errorf("line %d: unterminated string", l.line)
		}
		if l.peek() == '\\' {
			l.advance() // Skip escape character
		}
		l.advance()
	}
	if l.isAtEnd() {
		return fmt.Errorf("line %d: unterminated string", l.line)
	}
	l.advance() // Closing quote
	l.addTokenValue(TOKEN_STRING, l.source[l.start:l.current])
	return nil
}

func (l *Lexer) multilineString() error {
	// Already consumed first '['
	// Count equals signs
	eqCount := 0
	for l.peek() == '=' {
		l.advance()
		eqCount++
	}
	if l.peek() != '[' {
		// Not a valid multiline string, just a bracket
		l.current = l.start + 1
		l.addToken(TOKEN_LBRACKET)
		return nil
	}
	l.advance() // consume second '['

	startLine := l.line
	// Find matching ]=*]
	for !l.isAtEnd() {
		if l.peek() == '\n' {
			l.line++
			l.column = 0
		}
		if l.peek() == ']' {
			markPos := l.current
			l.advance()
			matchEq := 0
			for l.peek() == '=' && matchEq < eqCount {
				l.advance()
				matchEq++
			}
			if matchEq == eqCount && l.peek() == ']' {
				l.advance()
				l.addTokenValue(TOKEN_STRING, l.source[l.start:l.current])
				return nil
			}
			// Reset and continue from after the first ]
			l.current = markPos + 1
		} else {
			l.advance()
		}
	}
	return fmt.Errorf("line %d: unterminated multiline string (started at line %d)", l.line, startLine)
}

func (l *Lexer) templateString() error {
	// Consume everything in the template string, including ${...} interpolations
	// We'll store the raw content and parse interpolations later
	var builder strings.Builder
	builder.WriteByte('`')

	for l.peek() != '`' && !l.isAtEnd() {
		if l.peek() == '\n' {
			l.line++
			l.column = 0
		}
		if l.peek() == '\\' {
			builder.WriteByte(l.advance())
			if !l.isAtEnd() {
				builder.WriteByte(l.advance())
			}
		} else if l.peek() == '$' && l.peekNext() == '{' {
			builder.WriteByte(l.advance()) // $
			builder.WriteByte(l.advance()) // {
			braceDepth := 1
			for braceDepth > 0 && !l.isAtEnd() {
				c := l.advance()
				builder.WriteByte(c)
				if c == '{' {
					braceDepth++
				} else if c == '}' {
					braceDepth--
				} else if c == '\n' {
					l.line++
					l.column = 0
				}
			}
		} else {
			builder.WriteByte(l.advance())
		}
	}
	if l.isAtEnd() {
		return fmt.Errorf("line %d: unterminated template string", l.line)
	}
	l.advance() // Closing backtick
	builder.WriteByte('`')
	l.addTokenValue(TOKEN_TEMPLATE_STRING, builder.String())
	return nil
}

func (l *Lexer) number() {
	// Check for hex, binary, octal
	if l.source[l.start] == '0' && l.current < len(l.source) {
		switch l.peek() {
		case 'x', 'X':
			l.advance()
			for isHexDigit(l.peek()) {
				l.advance()
			}
			l.addToken(TOKEN_NUMBER)
			return
		case 'b', 'B':
			l.advance()
			for l.peek() == '0' || l.peek() == '1' {
				l.advance()
			}
			l.addToken(TOKEN_NUMBER)
			return
		case 'o', 'O':
			l.advance()
			for l.peek() >= '0' && l.peek() <= '7' {
				l.advance()
			}
			l.addToken(TOKEN_NUMBER)
			return
		}
	}

	// Regular decimal number
	for isDigit(l.peek()) {
		l.advance()
	}
	// Look for decimal part
	if l.peek() == '.' && isDigit(l.peekNext()) {
		l.advance() // consume '.'
		for isDigit(l.peek()) {
			l.advance()
		}
	}
	// Look for exponent
	if l.peek() == 'e' || l.peek() == 'E' {
		l.advance()
		if l.peek() == '+' || l.peek() == '-' {
			l.advance()
		}
		for isDigit(l.peek()) {
			l.advance()
		}
	}
	l.addToken(TOKEN_NUMBER)
}

func (l *Lexer) identifier() {
	for isAlphaNumeric(l.peek()) {
		l.advance()
	}
	text := l.source[l.start:l.current]
	tokenType, ok := keywords[text]
	if !ok {
		tokenType = TOKEN_IDENT
	}
	l.addToken(tokenType)
}

func (l *Lexer) advance() byte {
	c := l.source[l.current]
	l.current++
	l.column++
	return c
}

func (l *Lexer) match(expected byte) bool {
	if l.isAtEnd() || l.source[l.current] != expected {
		return false
	}
	l.current++
	l.column++
	return true
}

func (l *Lexer) peek() byte {
	if l.isAtEnd() {
		return 0
	}
	return l.source[l.current]
}

func (l *Lexer) peekNext() byte {
	if l.current+1 >= len(l.source) {
		return 0
	}
	return l.source[l.current+1]
}

func (l *Lexer) isAtEnd() bool {
	return l.current >= len(l.source)
}

func (l *Lexer) addToken(tokenType TokenType) {
	l.addTokenValue(tokenType, l.source[l.start:l.current])
}

func (l *Lexer) addTokenValue(tokenType TokenType, value string) {
	l.tokens = append(l.tokens, Token{
		Type:   tokenType,
		Value:  value,
		Line:   l.line,
		Column: l.startColumn,
	})
}

// Helper functions
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isAlphaNumeric(c byte) bool {
	return isAlpha(c) || isDigit(c)
}

// Convert tokimun number literals to Lua-compatible values
func ConvertNumber(value string) (string, error) {
	if len(value) < 2 {
		return value, nil
	}

	if value[0] == '0' {
		switch value[1] {
		case 'b', 'B':
			// Binary literal
			n, err := strconv.ParseInt(value[2:], 2, 64)
			if err != nil {
				return "", fmt.Errorf("invalid binary literal: %s", value)
			}
			return strconv.FormatInt(n, 10), nil
		case 'o', 'O':
			// Octal literal
			n, err := strconv.ParseInt(value[2:], 8, 64)
			if err != nil {
				return "", fmt.Errorf("invalid octal literal: %s", value)
			}
			return strconv.FormatInt(n, 10), nil
		case 'x', 'X':
			// Hex is already supported in Lua, pass through
			return value, nil
		}
	}
	return value, nil
}

// Check if a rune is a valid identifier start
func IsIdentifierStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}
