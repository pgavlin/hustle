package jq

import (
	"fmt"
	"strings"
)

// Precedence levels for Pratt-style parsing.
const (
	precPipe    = 1 // | (right-associative)
	precComma   = 2 // , (left-associative)
	precAs      = 3 // as (patterns)
	precAlt     = 4 // // (right-associative)
	precOr      = 5 // or
	precAnd     = 6 // and
	precCompare = 7 // == != < > <= >=
	precAdd     = 8 // + -
	precMul     = 9 // * / %
)

// Parse tokenizes src and parses it into an AST.
// Returns the root node and any parse errors encountered.
func Parse(src string) (Node, []string) {
	tokens := Tokenize(src)
	p := &parser{tokens: tokens}
	node := p.parseExpr(0)
	// If there are remaining non-EOF tokens, that's an error.
	if p.peek().Kind != TokenEOF {
		tok := p.peek()
		p.errorf("unexpected token %s", tok)
		// Consume remaining tokens
		for p.peek().Kind != TokenEOF {
			p.advance()
		}
	}
	return node, p.errors
}

type parser struct {
	tokens []Token
	pos    int
	errors []string
}

func (p *parser) peek() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	// Return an EOF token if past the end.
	last := p.tokens[len(p.tokens)-1]
	return Token{Kind: TokenEOF, Pos: last.End, End: last.End}
}

func (p *parser) advance() Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *parser) at(kinds ...TokenKind) bool {
	k := p.peek().Kind
	for _, kind := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

func (p *parser) atIdent(name string) bool {
	tok := p.peek()
	return tok.Kind == TokenIdent && tok.Value == name
}

func (p *parser) expect(kind TokenKind) Token {
	tok := p.peek()
	if tok.Kind == kind {
		return p.advance()
	}
	p.errorf("expected %s, got %s", tokenKindNames[kind], tok)
	return Token{Kind: kind, Pos: tok.Pos, End: tok.Pos}
}

func (p *parser) errorf(format string, args ...interface{}) {
	p.errors = append(p.errors, fmt.Sprintf(format, args...))
}

// parseExpr is the main Pratt-style expression parser.
func (p *parser) parseExpr(minPrec int) Node {
	left := p.parsePrimary()
	left = p.parsePostfix(left)

	for {
		tok := p.peek()
		prec, rightAssoc := p.infixInfo(tok)
		if prec < minPrec || prec == 0 {
			break
		}

		// Handle 'as' specially: expr as $pat | body
		if tok.Kind == TokenIdent && tok.Value == "as" && prec >= minPrec {
			p.advance() // consume 'as'
			patTok := p.expect(TokenVariable)
			p.expect(TokenPipe)
			body := p.parseExpr(precPipe)
			left = &AsNode{
				Span:    Span{Pos: left.nodeSpan().Pos, End: body.nodeSpan().End},
				Expr:    left,
				Pattern: patTok.Value,
				Body:    body,
			}
			continue
		}

		nextMinPrec := prec
		if !rightAssoc {
			nextMinPrec = prec + 1
		}

		switch tok.Kind {
		case TokenPipe:
			p.advance()
			right := p.parseExpr(nextMinPrec)
			left = &PipeNode{
				Span:  Span{Pos: left.nodeSpan().Pos, End: right.nodeSpan().End},
				Left:  left,
				Right: right,
			}
		case TokenComma:
			p.advance()
			right := p.parseExpr(nextMinPrec)
			left = &CommaNode{
				Span:  Span{Pos: left.nodeSpan().Pos, End: right.nodeSpan().End},
				Left:  left,
				Right: right,
			}
		default:
			op := p.advance()
			right := p.parseExpr(nextMinPrec)
			left = &BinOpNode{
				Span:  Span{Pos: left.nodeSpan().Pos, End: right.nodeSpan().End},
				Op:    opString(op),
				Left:  left,
				Right: right,
			}
		}

		left = p.parsePostfix(left)
	}

	return left
}

// infixInfo returns the precedence and right-associativity of a token
// when used as an infix operator. Returns (0, false) if not an infix operator.
func (p *parser) infixInfo(tok Token) (int, bool) {
	switch tok.Kind {
	case TokenPipe:
		return precPipe, true
	case TokenComma:
		return precComma, false
	case TokenAlt:
		return precAlt, true
	case TokenEq, TokenNe, TokenLt, TokenGt, TokenLe, TokenGe:
		return precCompare, false
	case TokenPlus, TokenMinus:
		return precAdd, false
	case TokenStar, TokenSlash, TokenPercent:
		return precMul, false
	case TokenIdent:
		switch tok.Value {
		case "or":
			return precOr, false
		case "and":
			return precAnd, false
		case "as":
			return precAs, true
		}
	}
	// Update operators as binary ops
	switch tok.Kind {
	case TokenUpdatePipe, TokenUpdateAdd, TokenUpdateSub, TokenUpdateMul,
		TokenUpdateDiv, TokenUpdateMod, TokenUpdateAlt, TokenAssign:
		return precPipe, true // update operators bind like pipe
	}
	return 0, false
}

func opString(tok Token) string {
	if tok.Value != "" {
		return tok.Value
	}
	if name, ok := tokenKindNames[tok.Kind]; ok {
		return name
	}
	return "?"
}

// parsePrimary parses a primary (atomic) expression.
func (p *parser) parsePrimary() Node {
	tok := p.peek()

	switch tok.Kind {
	case TokenDot:
		p.advance()
		return &IdentityNode{Span: Span{Pos: tok.Pos, End: tok.End}}

	case TokenDotField:
		p.advance()
		name := tok.Value[1:] // strip leading '.'
		return &FieldNode{Span: Span{Pos: tok.Pos, End: tok.End}, Name: name}

	case TokenDotDot:
		p.advance()
		return &RecurseNode{Span: Span{Pos: tok.Pos, End: tok.End}}

	case TokenLParen:
		return p.parseParen()

	case TokenLBracket:
		return p.parseArrayConstruction()

	case TokenLBrace:
		return p.parseObjectConstruction()

	case TokenString:
		p.advance()
		return &StringNode{
			Span:  Span{Pos: tok.Pos, End: tok.End},
			Value: unescapeString(tok.Value),
		}

	case TokenNumber:
		p.advance()
		return &NumberNode{Span: Span{Pos: tok.Pos, End: tok.End}, Value: tok.Value}

	case TokenVariable:
		p.advance()
		return &FuncNode{Span: Span{Pos: tok.Pos, End: tok.End}, Name: tok.Value}

	case TokenMinus:
		// Unary minus
		p.advance()
		expr := p.parsePrimary()
		expr = p.parsePostfix(expr)
		return &UnaryNode{
			Span: Span{Pos: tok.Pos, End: expr.nodeSpan().End},
			Op:   "-",
			Expr: expr,
		}

	case TokenIdent:
		return p.parseIdentPrimary()

	case TokenEOF:
		p.errorf("unexpected end of expression")
		return &IncompleteNode{Span: Span{Pos: tok.Pos, End: tok.End}, Token: "EOF"}

	default:
		p.errorf("unexpected token %s", tok)
		t := p.advance()
		return &IncompleteNode{Span: Span{Pos: t.Pos, End: t.End}, Token: t.String()}
	}
}

// parseIdentPrimary parses identifier-based primaries: keywords and function calls.
func (p *parser) parseIdentPrimary() Node {
	tok := p.peek()

	switch tok.Value {
	case "true":
		p.advance()
		return &BoolNode{Span: Span{Pos: tok.Pos, End: tok.End}, Value: true}
	case "false":
		p.advance()
		return &BoolNode{Span: Span{Pos: tok.Pos, End: tok.End}, Value: false}
	case "null":
		p.advance()
		return &NullNode{Span: Span{Pos: tok.Pos, End: tok.End}}
	case "if":
		return p.parseIf()
	case "try":
		return p.parseTry()
	case "reduce":
		return p.parseReduce()
	case "foreach":
		return p.parseForeach()
	case "label":
		return p.parseLabel()
	case "break":
		return p.parseBreak()
	case "def":
		return p.parseFuncDef()
	default:
		return p.parseFuncCall()
	}
}

// parseFuncCall parses a function call: name or name(arg; ...).
func (p *parser) parseFuncCall() Node {
	tok := p.advance() // consume the identifier
	name := tok.Value

	if !p.at(TokenLParen) {
		return &FuncNode{Span: Span{Pos: tok.Pos, End: tok.End}, Name: name}
	}

	// Parse arguments: name(arg1; arg2; ...)
	p.advance() // consume '('
	var args []Node
	if !p.at(TokenRParen, TokenEOF) {
		args = append(args, p.parseExpr(0))
		for p.at(TokenSemicolon) {
			p.advance()
			args = append(args, p.parseExpr(0))
		}
	}
	end := p.expect(TokenRParen)
	endPos := end.End
	if endPos == 0 && len(args) > 0 {
		endPos = args[len(args)-1].nodeSpan().End
	}

	return &FuncNode{
		Span: Span{Pos: tok.Pos, End: endPos},
		Name: name,
		Args: args,
	}
}

// parseParen parses (expr).
func (p *parser) parseParen() Node {
	open := p.advance() // consume '('
	expr := p.parseExpr(0)
	close := p.expect(TokenRParen)
	endPos := close.End
	if endPos == 0 {
		endPos = expr.nodeSpan().End
	}
	return &ParenNode{
		Span: Span{Pos: open.Pos, End: endPos},
		Expr: expr,
	}
}

// parseArrayConstruction parses [expr].
func (p *parser) parseArrayConstruction() Node {
	open := p.advance() // consume '['
	var expr Node
	if !p.at(TokenRBracket, TokenEOF) {
		expr = p.parseExpr(0)
	}
	close := p.expect(TokenRBracket)
	endPos := close.End
	if endPos == 0 {
		if expr != nil {
			endPos = expr.nodeSpan().End
		} else {
			endPos = open.End
		}
	}
	return &ArrayNode{
		Span: Span{Pos: open.Pos, End: endPos},
		Expr: expr,
	}
}

// parseObjectConstruction parses {k: v, ...}.
func (p *parser) parseObjectConstruction() Node {
	open := p.advance() // consume '{'
	var pairs []KeyValue

	for !p.at(TokenRBrace, TokenEOF) {
		if len(pairs) > 0 {
			p.expect(TokenComma)
			if p.at(TokenRBrace, TokenEOF) {
				break // trailing comma
			}
		}
		pair := p.parseObjectPair()
		pairs = append(pairs, pair)
	}

	close := p.expect(TokenRBrace)
	endPos := close.End
	if endPos == 0 {
		endPos = open.End
	}
	return &ObjectNode{
		Span:  Span{Pos: open.Pos, End: endPos},
		Pairs: pairs,
	}
}

// parseObjectPair parses one key: value pair in an object.
func (p *parser) parseObjectPair() KeyValue {
	tok := p.peek()

	switch {
	case tok.Kind == TokenIdent:
		// Bare identifier key: could be {foo} or {foo: expr}
		p.advance()
		key := &StringNode{
			Span:  Span{Pos: tok.Pos, End: tok.End},
			Value: tok.Value,
		}
		if p.at(TokenColon) {
			p.advance()
			val := p.parseExpr(precComma + 1)
			return KeyValue{Key: key, Value: val}
		}
		// Shorthand: {foo} means {foo: .foo}
		return KeyValue{Key: key, Value: nil}

	case tok.Kind == TokenString:
		p.advance()
		key := &StringNode{
			Span:  Span{Pos: tok.Pos, End: tok.End},
			Value: unescapeString(tok.Value),
		}
		if p.at(TokenColon) {
			p.advance()
			val := p.parseExpr(precComma + 1)
			return KeyValue{Key: key, Value: val}
		}
		return KeyValue{Key: key, Value: nil}

	case tok.Kind == TokenVariable:
		// {$var} shorthand or {$var: expr}
		p.advance()
		key := &StringNode{
			Span:  Span{Pos: tok.Pos, End: tok.End},
			Value: tok.Value,
		}
		if p.at(TokenColon) {
			p.advance()
			val := p.parseExpr(precComma + 1)
			return KeyValue{Key: key, Value: val}
		}
		return KeyValue{Key: key, Value: nil}

	case tok.Kind == TokenLParen:
		// Computed key: (expr)
		p.advance() // consume '('
		keyExpr := p.parseExpr(0)
		p.expect(TokenRParen)
		p.expect(TokenColon)
		val := p.parseExpr(precComma + 1)
		return KeyValue{Key: keyExpr, Value: val}

	case tok.Kind == TokenDotField:
		// .field as key shorthand
		p.advance()
		name := tok.Value[1:] // strip leading '.'
		key := &StringNode{
			Span:  Span{Pos: tok.Pos, End: tok.End},
			Value: name,
		}
		if p.at(TokenColon) {
			p.advance()
			val := p.parseExpr(precComma + 1)
			return KeyValue{Key: key, Value: val}
		}
		return KeyValue{Key: key, Value: nil}

	default:
		p.errorf("expected object key, got %s", tok)
		t := p.advance()
		return KeyValue{
			Key: &IncompleteNode{Span: Span{Pos: t.Pos, End: t.End}, Token: t.String()},
		}
	}
}

// parseIf parses if/then/elif/else/end.
func (p *parser) parseIf() Node {
	start := p.advance() // consume 'if'
	cond := p.parseExpr(0)
	p.expectIdent("then")
	then := p.parseExpr(0)

	var elifs []CondBody
	for p.atIdent("elif") {
		p.advance() // consume 'elif'
		elifCond := p.parseExpr(0)
		p.expectIdent("then")
		elifBody := p.parseExpr(0)
		elifs = append(elifs, CondBody{Cond: elifCond, Body: elifBody})
	}

	var elseNode Node
	if p.atIdent("else") {
		p.advance()
		elseNode = p.parseExpr(0)
	}

	end := p.expectIdent("end")
	endPos := end.End
	if endPos == 0 {
		if elseNode != nil {
			endPos = elseNode.nodeSpan().End
		} else {
			endPos = then.nodeSpan().End
		}
	}

	return &IfNode{
		Span:  Span{Pos: start.Pos, End: endPos},
		Cond:  cond,
		Then:  then,
		Elifs: elifs,
		Else:  elseNode,
	}
}

// parseTry parses try body [catch handler].
func (p *parser) parseTry() Node {
	start := p.advance() // consume 'try'
	body := p.parsePrimary()
	body = p.parsePostfix(body)

	var catch Node
	if p.atIdent("catch") {
		p.advance()
		catch = p.parsePrimary()
		catch = p.parsePostfix(catch)
	}

	endPos := body.nodeSpan().End
	if catch != nil {
		endPos = catch.nodeSpan().End
	}

	return &TryNode{
		Span:  Span{Pos: start.Pos, End: endPos},
		Body:  body,
		Catch: catch,
	}
}

// parseReduce parses: reduce expr as $pat (init; update)
func (p *parser) parseReduce() Node {
	start := p.advance() // consume 'reduce'
	expr := p.parseExpr(precAs + 1)
	p.expectIdent("as")
	pat := p.expect(TokenVariable)
	p.expect(TokenLParen)
	init := p.parseExpr(0)
	p.expect(TokenSemicolon)
	update := p.parseExpr(0)
	close := p.expect(TokenRParen)

	endPos := close.End
	if endPos == 0 {
		endPos = update.nodeSpan().End
	}

	return &ReduceNode{
		Span:    Span{Pos: start.Pos, End: endPos},
		Expr:    expr,
		Pattern: pat.Value,
		Init:    init,
		Update:  update,
	}
}

// parseForeach parses: foreach expr as $pat (init; update[; extract])
func (p *parser) parseForeach() Node {
	start := p.advance() // consume 'foreach'
	expr := p.parseExpr(precAs + 1)
	p.expectIdent("as")
	pat := p.expect(TokenVariable)
	p.expect(TokenLParen)
	init := p.parseExpr(0)
	p.expect(TokenSemicolon)
	update := p.parseExpr(0)

	var extract Node
	if p.at(TokenSemicolon) {
		p.advance()
		extract = p.parseExpr(0)
	}

	close := p.expect(TokenRParen)
	endPos := close.End
	if endPos == 0 {
		endPos = update.nodeSpan().End
		if extract != nil {
			endPos = extract.nodeSpan().End
		}
	}

	return &ForeachNode{
		Span:    Span{Pos: start.Pos, End: endPos},
		Expr:    expr,
		Pattern: pat.Value,
		Init:    init,
		Update:  update,
		Extract: extract,
	}
}

// parseLabel parses: label $name | body
func (p *parser) parseLabel() Node {
	start := p.advance() // consume 'label'
	name := p.expect(TokenVariable)
	p.expect(TokenPipe)
	body := p.parseExpr(precPipe)

	return &LabelNode{
		Span: Span{Pos: start.Pos, End: body.nodeSpan().End},
		Name: name.Value,
		Body: body,
	}
}

// parseBreak parses: break $name
func (p *parser) parseBreak() Node {
	start := p.advance() // consume 'break'
	name := p.expect(TokenVariable)
	endPos := name.End
	if endPos == 0 {
		endPos = start.End
	}
	return &BreakNode{
		Span: Span{Pos: start.Pos, End: endPos},
		Name: name.Value,
	}
}

// parseFuncDef parses: def name[(param; ...)]: body; rest
func (p *parser) parseFuncDef() Node {
	start := p.advance() // consume 'def'
	nameTok := p.expect(TokenIdent)
	name := nameTok.Value

	var params []string
	if p.at(TokenLParen) {
		p.advance() // consume '('
		for !p.at(TokenRParen, TokenEOF) {
			if len(params) > 0 {
				p.expect(TokenSemicolon)
			}
			paramTok := p.expect(TokenIdent)
			params = append(params, paramTok.Value)
		}
		p.expect(TokenRParen)
	}

	p.expect(TokenColon)
	body := p.parseExpr(0)
	p.expect(TokenSemicolon)

	// Parse the rest (the expression after the def)
	next := p.parseExpr(0)

	return &FuncDefNode{
		Span:   Span{Pos: start.Pos, End: next.nodeSpan().End},
		Name:   name,
		Params: params,
		Body:   body,
		Next:   next,
	}
}

// parsePostfix handles postfix operators after an expression.
func (p *parser) parsePostfix(expr Node) Node {
	for {
		tok := p.peek()
		switch tok.Kind {
		case TokenDotField:
			// .field suffix
			p.advance()
			name := tok.Value[1:] // strip leading '.'
			field := &FieldNode{Span: Span{Pos: tok.Pos, End: tok.End}, Name: name}
			expr = &SuffixNode{
				Span:   Span{Pos: expr.nodeSpan().Pos, End: tok.End},
				Left:   expr,
				Suffix: field,
			}

		case TokenDot:
			// Could be .["string"] postfix
			// Check if next is '[' — but only if no whitespace between dot and bracket
			// In jq, .["foo"] is a valid field access
			next := p.peekAt(1)
			if next.Kind == TokenLBracket && tok.End == next.Pos {
				p.advance() // consume '.'
				suffix := p.parseBracketSuffix()
				expr = &SuffixNode{
					Span:   Span{Pos: expr.nodeSpan().Pos, End: suffix.nodeSpan().End},
					Left:   expr,
					Suffix: suffix,
				}
			} else {
				return expr
			}

		case TokenLBracket:
			suffix := p.parseBracketSuffix()
			expr = &SuffixNode{
				Span:   Span{Pos: expr.nodeSpan().Pos, End: suffix.nodeSpan().End},
				Left:   expr,
				Suffix: suffix,
			}

		case TokenQuestion:
			p.advance()
			expr = &OptionalNode{
				Span: Span{Pos: expr.nodeSpan().Pos, End: tok.End},
				Expr: expr,
			}

		default:
			return expr
		}
	}
}

// parseBracketSuffix parses the bracket part: [], [expr], [expr:expr]
func (p *parser) parseBracketSuffix() Node {
	open := p.advance() // consume '['

	// Check for empty [] → IterNode
	if p.at(TokenRBracket) {
		close := p.advance()
		return &IterNode{Span: Span{Pos: open.Pos, End: close.End}}
	}

	// Check for [:expr] (slice with no low)
	if p.at(TokenColon) {
		p.advance() // consume ':'
		high := p.parseExpr(0)
		close := p.expect(TokenRBracket)
		endPos := close.End
		if endPos == 0 {
			endPos = high.nodeSpan().End
		}
		return &SliceNode{
			Span: Span{Pos: open.Pos, End: endPos},
			Low:  nil,
			High: high,
		}
	}

	// Parse the first expression
	expr := p.parseExpr(0)

	// Check for slice: [expr:expr]
	if p.at(TokenColon) {
		p.advance() // consume ':'
		var high Node
		if !p.at(TokenRBracket) {
			high = p.parseExpr(0)
		}
		close := p.expect(TokenRBracket)
		endPos := close.End
		if endPos == 0 {
			endPos = expr.nodeSpan().End
		}
		return &SliceNode{
			Span: Span{Pos: open.Pos, End: endPos},
			Low:  expr,
			High: high,
		}
	}

	// Simple index: [expr]
	close := p.expect(TokenRBracket)
	endPos := close.End
	if endPos == 0 {
		endPos = expr.nodeSpan().End
	}
	return &IndexNode{
		Span: Span{Pos: open.Pos, End: endPos},
		Expr: expr,
	}
}

// peekAt returns the token at offset n from current position.
func (p *parser) peekAt(n int) Token {
	idx := p.pos + n
	if idx < len(p.tokens) {
		return p.tokens[idx]
	}
	last := p.tokens[len(p.tokens)-1]
	return Token{Kind: TokenEOF, Pos: last.End, End: last.End}
}

// expectIdent expects an identifier with the given name.
func (p *parser) expectIdent(name string) Token {
	tok := p.peek()
	if tok.Kind == TokenIdent && tok.Value == name {
		return p.advance()
	}
	p.errorf("expected %q, got %s", name, tok)
	return Token{Kind: TokenIdent, Value: name, Pos: tok.Pos, End: tok.Pos}
}

// unescapeString strips outer quotes and unescapes a jq string literal.
func unescapeString(s string) string {
	if len(s) < 2 {
		return s
	}
	// Strip outer quotes
	if s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	} else if s[0] == '"' {
		s = s[1:]
	}

	if !strings.ContainsRune(s, '\\') {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case '/':
				b.WriteByte('/')
			default:
				// Unknown escape — preserve as-is
				b.WriteByte('\\')
				b.WriteByte(s[i])
			}
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
