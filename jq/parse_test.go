package jq

import "testing"

func TestParse_Identity(t *testing.T) {
	node, errs := Parse(".")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if _, ok := node.(*IdentityNode); !ok {
		t.Errorf("expected IdentityNode, got %T", node)
	}
}

func TestParse_Field(t *testing.T) {
	node, errs := Parse(".foo")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	f, ok := node.(*FieldNode)
	if !ok {
		t.Fatalf("expected FieldNode, got %T", node)
	}
	if f.Name != "foo" {
		t.Errorf("name = %q, want %q", f.Name, "foo")
	}
}

func TestParse_ChainedFields(t *testing.T) {
	node, errs := Parse(".foo.bar")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	// .foo.bar should be SuffixNode{Left: FieldNode{foo}, Suffix: FieldNode{bar}}
	s, ok := node.(*SuffixNode)
	if !ok {
		t.Fatalf("expected SuffixNode, got %T", node)
	}
	if _, ok := s.Left.(*FieldNode); !ok {
		t.Errorf("left should be FieldNode, got %T", s.Left)
	}
	if _, ok := s.Suffix.(*FieldNode); !ok {
		t.Errorf("suffix should be FieldNode, got %T", s.Suffix)
	}
}

func TestParse_Pipe(t *testing.T) {
	node, errs := Parse(".foo | .bar")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	p, ok := node.(*PipeNode)
	if !ok {
		t.Fatalf("expected PipeNode, got %T", node)
	}
	if _, ok := p.Left.(*FieldNode); !ok {
		t.Errorf("left should be FieldNode, got %T", p.Left)
	}
	if _, ok := p.Right.(*FieldNode); !ok {
		t.Errorf("right should be FieldNode, got %T", p.Right)
	}
}

func TestParse_Select(t *testing.T) {
	node, errs := Parse(`select(.level == "ERROR")`)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	f, ok := node.(*FuncNode)
	if !ok {
		t.Fatalf("expected FuncNode, got %T", node)
	}
	if f.Name != "select" {
		t.Errorf("name = %q, want %q", f.Name, "select")
	}
	if len(f.Args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(f.Args))
	}
}

func TestParse_Comparison(t *testing.T) {
	node, errs := Parse(`.level == "INFO"`)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	b, ok := node.(*BinOpNode)
	if !ok {
		t.Fatalf("expected BinOpNode, got %T", node)
	}
	if b.Op != "==" {
		t.Errorf("op = %q, want %q", b.Op, "==")
	}
}

func TestParse_Precedence_PipeVsAdd(t *testing.T) {
	// .a + .b | .c should be (.a + .b) | .c
	node, errs := Parse(".a + .b | .c")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	p, ok := node.(*PipeNode)
	if !ok {
		t.Fatalf("expected PipeNode at top, got %T", node)
	}
	if _, ok := p.Left.(*BinOpNode); !ok {
		t.Errorf("left of pipe should be BinOpNode, got %T", p.Left)
	}
}

func TestParse_IfThenElse(t *testing.T) {
	node, errs := Parse("if .x then .a else .b end")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if _, ok := node.(*IfNode); !ok {
		t.Fatalf("expected IfNode, got %T", node)
	}
}

func TestParse_ObjectConstruction(t *testing.T) {
	node, errs := Parse(`{a: .foo, b: .bar}`)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	obj, ok := node.(*ObjectNode)
	if !ok {
		t.Fatalf("expected ObjectNode, got %T", node)
	}
	if len(obj.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(obj.Pairs))
	}
}

func TestParse_ArrayConstruction(t *testing.T) {
	node, errs := Parse("[.foo, .bar]")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if _, ok := node.(*ArrayNode); !ok {
		t.Fatalf("expected ArrayNode, got %T", node)
	}
}

func TestParse_Iterator(t *testing.T) {
	node, errs := Parse(".[]")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	// .[] should parse as SuffixNode{Left: Identity, Suffix: Iter}
	s, ok := node.(*SuffixNode)
	if !ok {
		t.Fatalf("expected SuffixNode, got %T", node)
	}
	if _, ok := s.Suffix.(*IterNode); !ok {
		t.Errorf("suffix should be IterNode, got %T", s.Suffix)
	}
}

func TestParse_IndexAccess(t *testing.T) {
	node, errs := Parse(".[0]")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	s, ok := node.(*SuffixNode)
	if !ok {
		t.Fatalf("expected SuffixNode, got %T", node)
	}
	if _, ok := s.Suffix.(*IndexNode); !ok {
		t.Errorf("suffix should be IndexNode, got %T", s.Suffix)
	}
}

func TestParse_FuncDef(t *testing.T) {
	node, errs := Parse("def double: . * 2; double")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	fd, ok := node.(*FuncDefNode)
	if !ok {
		t.Fatalf("expected FuncDefNode, got %T", node)
	}
	if fd.Name != "double" {
		t.Errorf("name = %q, want %q", fd.Name, "double")
	}
}

func TestParse_Reduce(t *testing.T) {
	node, errs := Parse("reduce .[] as $x (0; . + $x)")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if _, ok := node.(*ReduceNode); !ok {
		t.Fatalf("expected ReduceNode, got %T", node)
	}
}

func TestParse_Try(t *testing.T) {
	node, errs := Parse("try .foo catch null")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	tr, ok := node.(*TryNode)
	if !ok {
		t.Fatalf("expected TryNode, got %T", node)
	}
	if tr.Catch == nil {
		t.Error("expected catch clause")
	}
}

// Error recovery tests

func TestParse_Incomplete_SelectDot(t *testing.T) {
	node, errs := Parse("select(.")
	// Should parse as FuncNode with an incomplete arg, with errors
	if len(errs) == 0 {
		t.Error("expected parse errors for incomplete expression")
	}
	f, ok := node.(*FuncNode)
	if !ok {
		t.Fatalf("expected FuncNode, got %T", node)
	}
	if f.Name != "select" {
		t.Errorf("name = %q, want select", f.Name)
	}
}

func TestParse_Incomplete_PipeRight(t *testing.T) {
	node, errs := Parse(".foo |")
	if len(errs) == 0 {
		t.Error("expected parse errors for incomplete pipe")
	}
	p, ok := node.(*PipeNode)
	if !ok {
		t.Fatalf("expected PipeNode, got %T", node)
	}
	if _, ok := p.Right.(*IncompleteNode); !ok {
		t.Errorf("right of pipe should be IncompleteNode, got %T", p.Right)
	}
}

func TestParse_Incomplete_DotAlone(t *testing.T) {
	// Just "." should parse fine as IdentityNode, not incomplete
	node, errs := Parse(".")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if _, ok := node.(*IdentityNode); !ok {
		t.Errorf("expected IdentityNode, got %T", node)
	}
}

func TestParse_SpanPositions(t *testing.T) {
	node, _ := Parse(".foo | .bar")
	p, ok := node.(*PipeNode)
	if !ok {
		t.Fatalf("expected PipeNode, got %T", node)
	}
	left := p.Left.nodeSpan()
	if left.Pos != 0 || left.End != 4 {
		t.Errorf(".foo span: %d-%d, want 0-4", left.Pos, left.End)
	}
	right := p.Right.nodeSpan()
	if right.Pos != 7 || right.End != 11 {
		t.Errorf(".bar span: %d-%d, want 7-11", right.Pos, right.End)
	}
}
