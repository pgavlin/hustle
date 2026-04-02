package jq

// Span tracks byte positions in source.
type Span struct {
	Pos int // start byte offset
	End int // end byte offset (exclusive)
}

// Node is the interface for all AST nodes.
type Node interface {
	nodeSpan() Span
}

// CondBody is an elif clause in an if expression.
type CondBody struct {
	Cond, Body Node
}

// KeyValue is a key-value pair in an object construction.
type KeyValue struct {
	Key   Node // StringNode, FieldNode, IdentityNode, or expression
	Value Node // nil means shorthand {foo} => {foo: .foo}
}

// PipeNode represents a | b.
type PipeNode struct {
	Span
	Left, Right Node
}

func (n *PipeNode) nodeSpan() Span { return n.Span }

// CommaNode represents a, b.
type CommaNode struct {
	Span
	Left, Right Node
}

func (n *CommaNode) nodeSpan() Span { return n.Span }

// IdentityNode represents the identity operator (.).
type IdentityNode struct {
	Span
}

func (n *IdentityNode) nodeSpan() Span { return n.Span }

// FieldNode represents .foo (field access).
type FieldNode struct {
	Span
	Name string
}

func (n *FieldNode) nodeSpan() Span { return n.Span }

// IndexNode represents .[expr].
type IndexNode struct {
	Span
	Expr Node
}

func (n *IndexNode) nodeSpan() Span { return n.Span }

// SliceNode represents .[low:high].
type SliceNode struct {
	Span
	Low, High Node
}

func (n *SliceNode) nodeSpan() Span { return n.Span }

// IterNode represents .[] (iterator).
type IterNode struct {
	Span
}

func (n *IterNode) nodeSpan() Span { return n.Span }

// RecurseNode represents .. (recursive descent).
type RecurseNode struct {
	Span
}

func (n *RecurseNode) nodeSpan() Span { return n.Span }

// FuncNode represents a function call: name or name(args; ...).
type FuncNode struct {
	Span
	Name string
	Args []Node
}

func (n *FuncNode) nodeSpan() Span { return n.Span }

// BinOpNode represents a binary operation: ==, +, and, or, //, etc.
type BinOpNode struct {
	Span
	Op          string
	Left, Right Node
}

func (n *BinOpNode) nodeSpan() Span { return n.Span }

// UnaryNode represents a unary operation: -expr.
type UnaryNode struct {
	Span
	Op   string
	Expr Node
}

func (n *UnaryNode) nodeSpan() Span { return n.Span }

// StringNode represents a string literal "hello".
type StringNode struct {
	Span
	Value string // unescaped content
}

func (n *StringNode) nodeSpan() Span { return n.Span }

// NumberNode represents a number literal: 42, 3.14.
type NumberNode struct {
	Span
	Value string
}

func (n *NumberNode) nodeSpan() Span { return n.Span }

// BoolNode represents true or false.
type BoolNode struct {
	Span
	Value bool
}

func (n *BoolNode) nodeSpan() Span { return n.Span }

// NullNode represents null.
type NullNode struct {
	Span
}

func (n *NullNode) nodeSpan() Span { return n.Span }

// IfNode represents if/then/elif/else/end.
type IfNode struct {
	Span
	Cond, Then Node
	Elifs      []CondBody
	Else       Node
}

func (n *IfNode) nodeSpan() Span { return n.Span }

// TryNode represents try/catch.
type TryNode struct {
	Span
	Body, Catch Node
}

func (n *TryNode) nodeSpan() Span { return n.Span }

// ReduceNode represents reduce expr as $pat (init; update).
type ReduceNode struct {
	Span
	Expr         Node
	Pattern      string
	Init, Update Node
}

func (n *ReduceNode) nodeSpan() Span { return n.Span }

// ForeachNode represents foreach expr as $pat (init; update; extract).
type ForeachNode struct {
	Span
	Expr            Node
	Pattern         string
	Init, Update    Node
	Extract         Node
}

func (n *ForeachNode) nodeSpan() Span { return n.Span }

// ObjectNode represents {k: v, ...}.
type ObjectNode struct {
	Span
	Pairs []KeyValue
}

func (n *ObjectNode) nodeSpan() Span { return n.Span }

// ArrayNode represents [expr].
type ArrayNode struct {
	Span
	Expr Node
}

func (n *ArrayNode) nodeSpan() Span { return n.Span }

// ParenNode represents (expr).
type ParenNode struct {
	Span
	Expr Node
}

func (n *ParenNode) nodeSpan() Span { return n.Span }

// OptionalNode represents expr?.
type OptionalNode struct {
	Span
	Expr Node
}

func (n *OptionalNode) nodeSpan() Span { return n.Span }

// SuffixNode represents chained access like .foo.bar[0]?.
type SuffixNode struct {
	Span
	Left   Node
	Suffix Node
}

func (n *SuffixNode) nodeSpan() Span { return n.Span }

// AsNode represents expr as $x | body.
type AsNode struct {
	Span
	Expr    Node
	Pattern string
	Body    Node
}

func (n *AsNode) nodeSpan() Span { return n.Span }

// LabelNode represents label $x | body.
type LabelNode struct {
	Span
	Name string
	Body Node
}

func (n *LabelNode) nodeSpan() Span { return n.Span }

// BreakNode represents break $x.
type BreakNode struct {
	Span
	Name string
}

func (n *BreakNode) nodeSpan() Span { return n.Span }

// FuncDefNode represents def f(a;b): body; rest.
type FuncDefNode struct {
	Span
	Name   string
	Params []string
	Body   Node
	Next   Node
}

func (n *FuncDefNode) nodeSpan() Span { return n.Span }

// IncompleteNode is a placeholder for error recovery.
type IncompleteNode struct {
	Span
	Token string
}

func (n *IncompleteNode) nodeSpan() Span { return n.Span }
