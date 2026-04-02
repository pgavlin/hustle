package jq

import (
	"unicode/utf8"
)

// Lexer tokenizes a jq expression string.
type Lexer struct {
	src    string
	pos    int
	tokens []Token
}

// Tokenize scans the entire source string and returns all tokens.
// The last token is always TokenEOF.
func Tokenize(src string) []Token {
	l := &Lexer{src: src}
	l.scan()
	return l.tokens
}

func (l *Lexer) scan() {
	for l.pos < len(l.src) {
		l.skipWhitespaceAndComments()
		if l.pos >= len(l.src) {
			break
		}
		start := l.pos
		ch := l.peek()
		switch {
		case ch == '.':
			l.scanDot(start)
		case ch == '$':
			l.scanVariable(start)
		case ch == '"':
			l.scanString(start)
		case ch >= '0' && ch <= '9':
			l.scanNumber(start)
		case isIdentStart(ch):
			l.scanIdent(start)
		default:
			l.scanOperator(start)
		}
	}
	l.tokens = append(l.tokens, Token{Kind: TokenEOF, Pos: l.pos, End: l.pos})
}

func (l *Lexer) peek() byte {
	if l.pos < len(l.src) {
		return l.src[l.pos]
	}
	return 0
}

func (l *Lexer) next() byte {
	ch := l.src[l.pos]
	l.pos++
	return ch
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.src) {
		ch := l.peek()
		if ch == '#' {
			// Skip to end of line
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.pos++
			}
		} else if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			l.pos++
		} else {
			break
		}
	}
}

func (l *Lexer) scanDot(start int) {
	l.pos++ // consume '.'

	if l.pos < len(l.src) {
		ch := l.peek()

		// ..
		if ch == '.' {
			l.pos++
			l.emit(TokenDotDot, start)
			return
		}

		// .field
		if isIdentStart(ch) {
			for l.pos < len(l.src) && isIdentPart(l.peek()) {
				l.pos++
			}
			l.tokens = append(l.tokens, Token{
				Kind:  TokenDotField,
				Value: l.src[start:l.pos],
				Pos:   start,
				End:   l.pos,
			})
			return
		}

		// .5 (number)
		if ch >= '0' && ch <= '9' {
			l.scanNumberFrom(start)
			return
		}
	}

	// Just '.'
	l.emit(TokenDot, start)
}

func (l *Lexer) scanVariable(start int) {
	l.pos++ // consume '$'
	for l.pos < len(l.src) && isIdentPart(l.peek()) {
		l.pos++
	}
	l.tokens = append(l.tokens, Token{
		Kind:  TokenVariable,
		Value: l.src[start:l.pos],
		Pos:   start,
		End:   l.pos,
	})
}

func (l *Lexer) scanString(start int) {
	l.pos++ // consume opening '"'
	for l.pos < len(l.src) {
		ch := l.next()
		if ch == '"' {
			l.tokens = append(l.tokens, Token{
				Kind:  TokenString,
				Value: l.src[start:l.pos],
				Pos:   start,
				End:   l.pos,
			})
			return
		}
		if ch == '\\' && l.pos < len(l.src) {
			l.pos++ // skip escaped character
		}
	}
	// Unterminated string — emit what we have
	l.tokens = append(l.tokens, Token{
		Kind:  TokenString,
		Value: l.src[start:l.pos],
		Pos:   start,
		End:   l.pos,
	})
}

func (l *Lexer) scanNumber(start int) {
	l.scanNumberFrom(start)
}

func (l *Lexer) scanNumberFrom(start int) {
	// Integer part (may already have consumed '.' for .5 case)
	for l.pos < len(l.src) && l.peek() >= '0' && l.peek() <= '9' {
		l.pos++
	}
	// Fractional part
	if l.pos < len(l.src) && l.peek() == '.' {
		next := l.pos + 1
		if next < len(l.src) && l.src[next] >= '0' && l.src[next] <= '9' {
			l.pos++ // consume '.'
			for l.pos < len(l.src) && l.peek() >= '0' && l.peek() <= '9' {
				l.pos++
			}
		}
	}
	// Exponent
	if l.pos < len(l.src) && (l.peek() == 'e' || l.peek() == 'E') {
		l.pos++
		if l.pos < len(l.src) && (l.peek() == '+' || l.peek() == '-') {
			l.pos++
		}
		for l.pos < len(l.src) && l.peek() >= '0' && l.peek() <= '9' {
			l.pos++
		}
	}
	l.tokens = append(l.tokens, Token{
		Kind:  TokenNumber,
		Value: l.src[start:l.pos],
		Pos:   start,
		End:   l.pos,
	})
}

func (l *Lexer) scanIdent(start int) {
	for l.pos < len(l.src) && isIdentPart(l.peek()) {
		l.pos++
	}
	l.tokens = append(l.tokens, Token{
		Kind:  TokenIdent,
		Value: l.src[start:l.pos],
		Pos:   start,
		End:   l.pos,
	})
}

func (l *Lexer) scanOperator(start int) {
	ch := l.next()
	switch ch {
	case '|':
		if l.pos < len(l.src) && l.peek() == '=' {
			l.pos++
			l.emit(TokenUpdatePipe, start)
		} else {
			l.emit(TokenPipe, start)
		}
	case ',':
		l.emit(TokenComma, start)
	case '+':
		if l.pos < len(l.src) && l.peek() == '=' {
			l.pos++
			l.emit(TokenUpdateAdd, start)
		} else {
			l.emit(TokenPlus, start)
		}
	case '-':
		if l.pos < len(l.src) && l.peek() == '=' {
			l.pos++
			l.emit(TokenUpdateSub, start)
		} else {
			l.emit(TokenMinus, start)
		}
	case '*':
		if l.pos < len(l.src) && l.peek() == '=' {
			l.pos++
			l.emit(TokenUpdateMul, start)
		} else {
			l.emit(TokenStar, start)
		}
	case '/':
		if l.pos < len(l.src) && l.peek() == '/' {
			l.pos++
			if l.pos < len(l.src) && l.peek() == '=' {
				l.pos++
				l.emit(TokenUpdateAlt, start)
			} else {
				l.emit(TokenAlt, start)
			}
		} else if l.pos < len(l.src) && l.peek() == '=' {
			l.pos++
			l.emit(TokenUpdateDiv, start)
		} else {
			l.emit(TokenSlash, start)
		}
	case '%':
		if l.pos < len(l.src) && l.peek() == '=' {
			l.pos++
			l.emit(TokenUpdateMod, start)
		} else {
			l.emit(TokenPercent, start)
		}
	case '=':
		if l.pos < len(l.src) && l.peek() == '=' {
			l.pos++
			l.emit(TokenEq, start)
		} else {
			l.emit(TokenAssign, start)
		}
	case '!':
		if l.pos < len(l.src) && l.peek() == '=' {
			l.pos++
			l.emit(TokenNe, start)
		} else {
			// invalid character — emit as a single-char token for error recovery
			l.tokens = append(l.tokens, Token{
				Kind:  TokenIdent,
				Value: "!",
				Pos:   start,
				End:   l.pos,
			})
		}
	case '<':
		if l.pos < len(l.src) && l.peek() == '=' {
			l.pos++
			l.emit(TokenLe, start)
		} else {
			l.emit(TokenLt, start)
		}
	case '>':
		if l.pos < len(l.src) && l.peek() == '=' {
			l.pos++
			l.emit(TokenGe, start)
		} else {
			l.emit(TokenGt, start)
		}
	case '(':
		l.emit(TokenLParen, start)
	case ')':
		l.emit(TokenRParen, start)
	case '[':
		l.emit(TokenLBracket, start)
	case ']':
		l.emit(TokenRBracket, start)
	case '{':
		l.emit(TokenLBrace, start)
	case '}':
		l.emit(TokenRBrace, start)
	case ':':
		l.emit(TokenColon, start)
	case ';':
		l.emit(TokenSemicolon, start)
	case '?':
		l.emit(TokenQuestion, start)
	default:
		// Skip unknown characters (for error tolerance)
		// Try to handle multi-byte UTF-8 characters
		if ch >= 0x80 {
			// Back up and read as rune
			l.pos = start
			_, size := utf8.DecodeRuneInString(l.src[l.pos:])
			l.pos += size
		}
	}
}

func (l *Lexer) emit(kind TokenKind, start int) {
	l.tokens = append(l.tokens, Token{
		Kind:  kind,
		Value: l.src[start:l.pos],
		Pos:   start,
		End:   l.pos,
	})
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}
