package jq

import "sort"

// completableBuiltins lists jq builtins for tab completion.
var completableBuiltins = []string{
	"select(", "empty", "error(",
	"not", "type", "length", "keys", "values", "has(", "contains(",
	"to_entries", "from_entries", "with_entries(",
	"test(", "match(", "startswith(", "endswith(",
	"split(", "join(", "ascii_downcase", "ascii_upcase",
	"ltrimstr(", "rtrimstr(", "tostring", "tonumber",
	"map(", "map_values(", "sort", "sort_by(", "reverse",
	"group_by(", "unique", "unique_by(", "flatten",
	"first", "last", "range(", "limit(",
	"add", "any", "all", "min", "max", "min_by(", "max_by(",
	"tojson", "fromjson",
	"null", "true", "false",
	"if ", "reduce ", "foreach ", "try ", "def ",
	"abs", "floor", "ceil", "round",
	"recurse", "debug", "path(",
	"objects", "arrays", "strings", "numbers", "booleans", "nulls",
}

// Complete returns tab-completion suggestions for a jq expression at the given
// cursor position, using symbolic evaluation to determine available fields.
func Complete(expr string, cursorPos int, inputShape Shape) []string {
	if cursorPos > len(expr) {
		cursorPos = len(expr)
	}
	before := expr[:cursorPos]

	// Find the completion token by scanning backwards from cursor
	tokenStart, token := findToken(before)

	// Determine the shape at the cursor position
	shape := shapeAtCursor(expr, cursorPos, inputShape)

	// Generate suggestions
	var suggestions []string
	prefix := expr[:tokenStart]

	if token == "" {
		// Empty — suggest both dot-fields and builtins
		for _, name := range FieldNames(shape) {
			suggestions = append(suggestions, prefix+"."+name)
		}
		for _, b := range completableBuiltins {
			suggestions = append(suggestions, prefix+b)
		}
	} else if token[0] == '.' {
		// Field completion
		fieldPrefix := token[1:] // strip the leading dot
		var names []string
		if names = FieldNames(shape); names == nil {
			// AnyShape fallback: use inputShape
			names = FieldNames(inputShape)
		}
		for _, name := range names {
			if fieldPrefix == "" || hasPrefix(name, fieldPrefix) {
				suggestions = append(suggestions, prefix+"."+name)
			}
		}
	} else {
		// Builtin completion
		for _, b := range completableBuiltins {
			if hasPrefix(b, token) {
				suggestions = append(suggestions, prefix+b)
			}
		}
	}

	sort.Strings(suggestions)
	return suggestions
}

// findToken scans backwards from the end of `before` to find the completion token.
// Returns (tokenStart, token) where tokenStart is the byte offset in before.
func findToken(before string) (int, string) {
	if len(before) == 0 {
		return 0, ""
	}

	i := len(before) - 1

	// Check if we're in a dot-field context
	// Scan back through identifier chars
	for i >= 0 && isTokenChar(before[i]) {
		i--
	}

	// If we stopped at a dot, include it (it's a field access)
	if i >= 0 && before[i] == '.' {
		return i, before[i:]
	}

	// If we scanned back some identifier chars but no dot, it's a bare word (builtin)
	if i+1 < len(before) {
		return i + 1, before[i+1:]
	}

	// Check if cursor is right after a dot
	if before[len(before)-1] == '.' {
		return len(before) - 1, "."
	}

	return len(before), ""
}

func isTokenChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_' || ch == '-'
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// shapeAtCursor determines the Shape of the jq value (.) at the cursor position
// by parsing the expression and symbolically evaluating up to the cursor.
func shapeAtCursor(expr string, cursorPos int, inputShape Shape) Shape {
	node, _ := Parse(expr)
	if node == nil {
		return inputShape
	}
	return evalShapeAtCursor(node, inputShape, cursorPos)
}

// evalShapeAtCursor walks the AST to find what shape flows into the cursor position.
func evalShapeAtCursor(node Node, input Shape, cursor int) Shape {
	switch n := node.(type) {
	case *PipeNode:
		rightSpan := n.Right.nodeSpan()
		if cursor >= rightSpan.Pos {
			// Cursor is in the right side of the pipe.
			// Evaluate the left side to get the shape flowing into the right.
			leftShape := symbolicEvalShape(n.Left, input)
			return evalShapeAtCursor(n.Right, leftShape, cursor)
		}
		return evalShapeAtCursor(n.Left, input, cursor)

	case *CommaNode:
		rightSpan := n.Right.nodeSpan()
		if cursor >= rightSpan.Pos {
			return evalShapeAtCursor(n.Right, input, cursor)
		}
		return evalShapeAtCursor(n.Left, input, cursor)

	case *FuncNode:
		// Inside a function call, args receive the same input shape
		for _, arg := range n.Args {
			span := arg.nodeSpan()
			if cursor >= span.Pos && cursor <= span.End {
				return evalShapeAtCursor(arg, input, cursor)
			}
		}
		return input

	case *SuffixNode:
		suffixSpan := n.Suffix.nodeSpan()
		if cursor >= suffixSpan.Pos {
			// Cursor is in the suffix — evaluate Left to get the shape
			leftShape := symbolicEvalShape(n.Left, input)
			return evalShapeAtCursor(n.Suffix, leftShape, cursor)
		}
		return evalShapeAtCursor(n.Left, input, cursor)

	case *ParenNode:
		if n.Expr != nil {
			return evalShapeAtCursor(n.Expr, input, cursor)
		}
		return input

	case *IfNode:
		// Check which part the cursor is in
		if n.Then != nil {
			thenSpan := n.Then.nodeSpan()
			if cursor >= thenSpan.Pos && cursor <= thenSpan.End {
				return evalShapeAtCursor(n.Then, input, cursor)
			}
		}
		if n.Else != nil {
			elseSpan := n.Else.nodeSpan()
			if cursor >= elseSpan.Pos && cursor <= elseSpan.End {
				return evalShapeAtCursor(n.Else, input, cursor)
			}
		}
		if n.Cond != nil {
			return evalShapeAtCursor(n.Cond, input, cursor)
		}
		return input

	case *ArrayNode:
		if n.Expr != nil {
			return evalShapeAtCursor(n.Expr, input, cursor)
		}
		return input

	case *BinOpNode:
		rightSpan := n.Right.nodeSpan()
		if cursor >= rightSpan.Pos {
			return evalShapeAtCursor(n.Right, input, cursor)
		}
		return evalShapeAtCursor(n.Left, input, cursor)

	case *TryNode:
		if n.Body != nil {
			return evalShapeAtCursor(n.Body, input, cursor)
		}
		return input

	case *FuncDefNode:
		if n.Next != nil {
			nextSpan := n.Next.nodeSpan()
			if cursor >= nextSpan.Pos {
				return evalShapeAtCursor(n.Next, input, cursor)
			}
		}
		return input

	default:
		// For leaf nodes or nodes we don't recurse into, return the input shape
		return input
	}
}

// symbolicEvalShape evaluates a node symbolically and returns the output shape.
func symbolicEvalShape(node Node, input Shape) Shape {
	sym := SymbolicValue(input)
	for v := range Eval(node, sym, DefaultEnv()) {
		return v.Shape
	}
	return AnyShape{}
}
