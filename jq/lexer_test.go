package jq

import "testing"

func tokenKinds(tokens []Token) []TokenKind {
	kinds := make([]TokenKind, len(tokens))
	for i, t := range tokens {
		kinds[i] = t.Kind
	}
	return kinds
}

func TestTokenize_Identity(t *testing.T) {
	tokens := Tokenize(".")
	if len(tokens) != 2 || tokens[0].Kind != TokenDot {
		t.Errorf("expected [Dot EOF], got %v", tokenKinds(tokens))
	}
}

func TestTokenize_DotField(t *testing.T) {
	tokens := Tokenize(".foo")
	if len(tokens) != 2 || tokens[0].Kind != TokenDotField || tokens[0].Value != ".foo" {
		t.Errorf("expected [DotField(.foo) EOF], got %v %q", tokenKinds(tokens), tokens[0].Value)
	}
}

func TestTokenize_DotDot(t *testing.T) {
	tokens := Tokenize("..")
	if len(tokens) != 2 || tokens[0].Kind != TokenDotDot {
		t.Errorf("expected [DotDot EOF], got %v", tokenKinds(tokens))
	}
}

func TestTokenize_ChainedFields(t *testing.T) {
	tokens := Tokenize(".foo.bar")
	// Should be: .foo . bar? No — .foo is one token, then .bar is another
	// Actually .foo.bar should tokenize as .foo then .bar
	// The lexer handles .foo as TokenDotField. After .foo, the next char is '.'
	// which starts a new dot scan → .bar is another TokenDotField
	if len(tokens) != 3 { // .foo, .bar, EOF
		t.Errorf("expected 3 tokens, got %d: %v", len(tokens), tokenKinds(tokens))
	}
	if tokens[0].Kind != TokenDotField || tokens[0].Value != ".foo" {
		t.Errorf("token 0: expected .foo, got %v %q", tokens[0].Kind, tokens[0].Value)
	}
	if tokens[1].Kind != TokenDotField || tokens[1].Value != ".bar" {
		t.Errorf("token 1: expected .bar, got %v %q", tokens[1].Kind, tokens[1].Value)
	}
}

func TestTokenize_Pipe(t *testing.T) {
	tokens := Tokenize(".foo | .bar")
	if len(tokens) != 4 { // .foo | .bar EOF
		t.Errorf("expected 4 tokens, got %d", len(tokens))
	}
	if tokens[1].Kind != TokenPipe {
		t.Errorf("expected Pipe, got %v", tokens[1].Kind)
	}
}

func TestTokenize_Select(t *testing.T) {
	tokens := Tokenize(`select(.level == "ERROR")`)
	// select ( .level == "ERROR" ) EOF
	if len(tokens) != 7 {
		t.Errorf("expected 7 tokens, got %d: %v", len(tokens), tokenKinds(tokens))
	}
	if tokens[0].Kind != TokenIdent || tokens[0].Value != "select" {
		t.Errorf("expected select, got %v %q", tokens[0].Kind, tokens[0].Value)
	}
	if tokens[3].Kind != TokenEq {
		t.Errorf("expected ==, got %v", tokens[3].Kind)
	}
	if tokens[4].Kind != TokenString {
		t.Errorf("expected string, got %v", tokens[4].Kind)
	}
}

func TestTokenize_Number(t *testing.T) {
	tokens := Tokenize("42 3.14 1e10")
	if len(tokens) != 4 { // 42 3.14 1e10 EOF
		t.Errorf("expected 4 tokens, got %d", len(tokens))
	}
	for i := 0; i < 3; i++ {
		if tokens[i].Kind != TokenNumber {
			t.Errorf("token %d: expected Number, got %v", i, tokens[i].Kind)
		}
	}
	if tokens[0].Value != "42" || tokens[1].Value != "3.14" || tokens[2].Value != "1e10" {
		t.Errorf("unexpected values: %q %q %q", tokens[0].Value, tokens[1].Value, tokens[2].Value)
	}
}

func TestTokenize_DotNumber(t *testing.T) {
	tokens := Tokenize(".5")
	if len(tokens) != 2 || tokens[0].Kind != TokenNumber || tokens[0].Value != ".5" {
		t.Errorf("expected [Number(.5) EOF], got %v %q", tokenKinds(tokens), tokens[0].Value)
	}
}

func TestTokenize_String(t *testing.T) {
	tokens := Tokenize(`"hello \"world\""`)
	if len(tokens) != 2 || tokens[0].Kind != TokenString {
		t.Errorf("expected [String EOF], got %v", tokenKinds(tokens))
	}
}

func TestTokenize_Variable(t *testing.T) {
	tokens := Tokenize("$x")
	if len(tokens) != 2 || tokens[0].Kind != TokenVariable || tokens[0].Value != "$x" {
		t.Errorf("expected [Variable($x) EOF], got %v %q", tokenKinds(tokens), tokens[0].Value)
	}
}

func TestTokenize_Comparison(t *testing.T) {
	tokens := Tokenize("!= <= >= == < >")
	kinds := tokenKinds(tokens)
	expected := []TokenKind{TokenNe, TokenLe, TokenGe, TokenEq, TokenLt, TokenGt, TokenEOF}
	if len(kinds) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(kinds), kinds)
	}
	for i := range expected {
		if kinds[i] != expected[i] {
			t.Errorf("token %d: expected %v, got %v", i, expected[i], kinds[i])
		}
	}
}

func TestTokenize_AltOperator(t *testing.T) {
	tokens := Tokenize(". // null")
	if len(tokens) != 4 { // . // null EOF
		t.Errorf("expected 4 tokens, got %d: %v", len(tokens), tokenKinds(tokens))
	}
	if tokens[1].Kind != TokenAlt {
		t.Errorf("expected Alt, got %v", tokens[1].Kind)
	}
}

func TestTokenize_UpdateOperators(t *testing.T) {
	tokens := Tokenize("|= += -= *= /= %= //=")
	expected := []TokenKind{TokenUpdatePipe, TokenUpdateAdd, TokenUpdateSub, TokenUpdateMul, TokenUpdateDiv, TokenUpdateMod, TokenUpdateAlt, TokenEOF}
	kinds := tokenKinds(tokens)
	if len(kinds) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(kinds), kinds)
	}
	for i := range expected {
		if kinds[i] != expected[i] {
			t.Errorf("token %d: expected %v, got %v", i, expected[i], kinds[i])
		}
	}
}

func TestTokenize_Comment(t *testing.T) {
	tokens := Tokenize(".foo # comment\n| .bar")
	if len(tokens) != 4 { // .foo | .bar EOF
		t.Errorf("expected 4 tokens, got %d: %v", len(tokens), tokenKinds(tokens))
	}
}

func TestTokenize_Positions(t *testing.T) {
	tokens := Tokenize(".foo | .bar")
	// .foo at 0-4, | at 5-6, .bar at 7-11
	if tokens[0].Pos != 0 || tokens[0].End != 4 {
		t.Errorf(".foo: pos=%d end=%d, want 0-4", tokens[0].Pos, tokens[0].End)
	}
	if tokens[1].Pos != 5 || tokens[1].End != 6 {
		t.Errorf("|: pos=%d end=%d, want 5-6", tokens[1].Pos, tokens[1].End)
	}
	if tokens[2].Pos != 7 || tokens[2].End != 11 {
		t.Errorf(".bar: pos=%d end=%d, want 7-11", tokens[2].Pos, tokens[2].End)
	}
}

func TestTokenize_Brackets(t *testing.T) {
	tokens := Tokenize(".[] | .[0]")
	// . [ ] | . [ 0 ] EOF
	if len(tokens) != 9 {
		t.Errorf("expected 9 tokens, got %d: %v", len(tokens), tokenKinds(tokens))
	}
}

func TestTokenize_Object(t *testing.T) {
	tokens := Tokenize(`{a: .foo, b: .bar}`)
	// { a : .foo , b : .bar } EOF
	if tokens[0].Kind != TokenLBrace {
		t.Errorf("expected LBrace, got %v", tokens[0].Kind)
	}
}

func TestTokenize_Empty(t *testing.T) {
	tokens := Tokenize("")
	if len(tokens) != 1 || tokens[0].Kind != TokenEOF {
		t.Errorf("expected [EOF], got %v", tokenKinds(tokens))
	}
}
