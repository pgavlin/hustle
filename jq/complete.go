package jq

import (
	"fmt"
	"sort"
	"strings"
)

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

// Suggestion is a completion suggestion with optional builtin metadata.
type Suggestion struct {
	Text    string // full expression text for textinput
	Builtin string // builtin name if this is a builtin suggestion, else ""
}

// completionContext describes what's available for completion at a cursor position.
type completionContext struct {
	shape      Shape // shape of . at cursor
	valueShape Shape // if in comparison RHS, shape of the LHS (may have EnumValues)
}

// Complete returns tab-completion suggestions for a jq expression at the given
// cursor position, using symbolic evaluation to determine available fields.
func Complete(expr string, cursorPos int, inputShape Shape) []Suggestion {
	if cursorPos > len(expr) {
		cursorPos = len(expr)
	}
	before := expr[:cursorPos]

	// Find the completion token by scanning backwards from cursor
	tokenStart, token := findToken(before)

	// Determine the completion context
	ctx := contextAtCursor(expr, cursorPos, inputShape)

	// Generate suggestions
	var suggestions []Suggestion
	prefix := expr[:tokenStart]

	if token == "" {
		// Check if we're in a value position (RHS of comparison) with enum values.
		// But don't suggest if cursor is right after a closing quote (value already entered).
		rightAfterQuote := cursorPos > 0 && expr[cursorPos-1] == '"'
		if ctx.valueShape != nil && !rightAfterQuote {
			if vals := EnumValues(ctx.valueShape); vals != nil {
				for _, v := range vals {
					suggestions = append(suggestions, Suggestion{Text: prefix + formatValue(v)})
				}
				sort.Slice(suggestions, func(i, j int) bool {
					return suggestions[i].Text < suggestions[j].Text
				})
				return suggestions
			}
		}
		// Empty — suggest both dot-fields and builtins
		for _, name := range FieldNames(ctx.shape) {
			suggestions = append(suggestions, Suggestion{Text: prefix + "." + name})
		}
		for _, b := range completableBuiltins {
			suggestions = append(suggestions, Suggestion{
				Text:    prefix + b,
				Builtin: builtinName(b),
			})
		}
	} else if token == `"` || (len(token) > 0 && token[0] == '"') {
		// Partial string in value position — suggest enum values matching prefix
		if ctx.valueShape != nil {
			if vals := EnumValues(ctx.valueShape); vals != nil {
				for _, v := range vals {
					formatted := formatValue(v)
					if hasPrefix(formatted, token) {
						suggestions = append(suggestions, Suggestion{Text: prefix + formatted})
					}
				}
			}
		}
	} else if token[0] == '.' {
		// Field completion
		fieldPrefix := token[1:] // strip the leading dot
		var names []string
		if names = FieldNames(ctx.shape); names == nil {
			// AnyShape fallback: use inputShape
			names = FieldNames(inputShape)
		}
		for _, name := range names {
			if fieldPrefix == "" || hasPrefix(name, fieldPrefix) {
				suggestions = append(suggestions, Suggestion{Text: prefix + "." + name})
			}
		}
	} else {
		// Builtin completion
		for _, b := range completableBuiltins {
			if hasPrefix(b, token) {
				suggestions = append(suggestions, Suggestion{
					Text:    prefix + b,
					Builtin: builtinName(b),
				})
			}
		}
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Text < suggestions[j].Text
	})
	return suggestions
}

// formatValue formats a value for insertion into a jq expression.
func formatValue(v any) string {
	switch v := v.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	default:
		return fmt.Sprint(v)
	}
}

// builtinName strips the trailing "(" or " " from a completable builtin string.
func builtinName(b string) string {
	return strings.TrimRight(b, "( ")
}

// findToken scans backwards from the end of `before` to find the completion token.
// Returns (tokenStart, token) where tokenStart is the byte offset in before.
func findToken(before string) (int, string) {
	if len(before) == 0 {
		return 0, ""
	}

	i := len(before) - 1

	// Check for partial string literal by counting unescaped quotes.
	// If odd, we're inside an unclosed string — the token is from the opening quote.
	quoteCount := 0
	lastQuote := -1
	for k := 0; k < len(before); k++ {
		if before[k] == '"' && (k == 0 || before[k-1] != '\\') {
			quoteCount++
			lastQuote = k
		}
	}
	if quoteCount%2 == 1 && lastQuote >= 0 {
		// Odd number of quotes — we're inside an unclosed string
		return lastQuote, before[lastQuote:]
	}

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

// contextAtCursor determines the completion context at the cursor position.
func contextAtCursor(expr string, cursorPos int, inputShape Shape) completionContext {
	node, _ := Parse(expr)
	if node == nil {
		return completionContext{shape: inputShape}
	}
	return evalContextAtCursor(node, inputShape, cursorPos)
}

// evalContextAtCursor walks the AST to find what shape flows into the cursor position
// and whether we're in a comparison value position.
func evalContextAtCursor(node Node, input Shape, cursor int) completionContext {
	switch n := node.(type) {
	case *PipeNode:
		rightSpan := n.Right.nodeSpan()
		if cursor >= rightSpan.Pos {
			leftShape := symbolicEvalShape(n.Left, input)
			return evalContextAtCursor(n.Right, leftShape, cursor)
		}
		return evalContextAtCursor(n.Left, input, cursor)

	case *CommaNode:
		rightSpan := n.Right.nodeSpan()
		if cursor >= rightSpan.Pos {
			return evalContextAtCursor(n.Right, input, cursor)
		}
		return evalContextAtCursor(n.Left, input, cursor)

	case *FuncNode:
		for _, arg := range n.Args {
			span := arg.nodeSpan()
			if cursor >= span.Pos && cursor <= span.End {
				return evalContextAtCursor(arg, input, cursor)
			}
		}
		return completionContext{shape: input}

	case *SuffixNode:
		suffixSpan := n.Suffix.nodeSpan()
		if cursor >= suffixSpan.Pos {
			leftShape := symbolicEvalShape(n.Left, input)
			return evalContextAtCursor(n.Suffix, leftShape, cursor)
		}
		return evalContextAtCursor(n.Left, input, cursor)

	case *ParenNode:
		if n.Expr != nil {
			return evalContextAtCursor(n.Expr, input, cursor)
		}
		return completionContext{shape: input}

	case *IfNode:
		if n.Then != nil {
			thenSpan := n.Then.nodeSpan()
			if cursor >= thenSpan.Pos && cursor <= thenSpan.End {
				return evalContextAtCursor(n.Then, input, cursor)
			}
		}
		if n.Else != nil {
			elseSpan := n.Else.nodeSpan()
			if cursor >= elseSpan.Pos && cursor <= elseSpan.End {
				return evalContextAtCursor(n.Else, input, cursor)
			}
		}
		if n.Cond != nil {
			return evalContextAtCursor(n.Cond, input, cursor)
		}
		return completionContext{shape: input}

	case *ArrayNode:
		if n.Expr != nil {
			return evalContextAtCursor(n.Expr, input, cursor)
		}
		return completionContext{shape: input}

	case *BinOpNode:
		rightSpan := n.Right.nodeSpan()
		if cursor >= rightSpan.Pos {
			// Cursor is on the RHS of a binary operator
			if isComparisonOp(n.Op) {
				// Evaluate the LHS to get its shape (which may carry enum values)
				lhsShape := symbolicEvalShape(n.Left, input)
				return completionContext{
					shape:      input,
					valueShape: lhsShape,
				}
			}
			return evalContextAtCursor(n.Right, input, cursor)
		}
		return evalContextAtCursor(n.Left, input, cursor)

	case *TryNode:
		if n.Body != nil {
			return evalContextAtCursor(n.Body, input, cursor)
		}
		return completionContext{shape: input}

	case *FuncDefNode:
		if n.Next != nil {
			nextSpan := n.Next.nodeSpan()
			if cursor >= nextSpan.Pos {
				return evalContextAtCursor(n.Next, input, cursor)
			}
		}
		return completionContext{shape: input}

	default:
		return completionContext{shape: input}
	}
}

func isComparisonOp(op string) bool {
	switch op {
	case "==", "!=", "<", ">", "<=", ">=":
		return true
	}
	return false
}

// symbolicEvalShape evaluates a node symbolically and returns the output shape.
func symbolicEvalShape(node Node, input Shape) Shape {
	sym := SymbolicValue(input)
	for v := range Eval(node, sym, DefaultEnv()) {
		return v.Shape
	}
	return AnyShape{}
}
