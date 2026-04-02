package jq

import (
	"iter"
	"testing"
)

// Helper to collect all values from an iterator
func collect(seq iter.Seq[Value]) []Value {
	var vals []Value
	for v := range seq {
		vals = append(vals, v)
	}
	return vals
}

// Helper to get first value
func first(seq iter.Seq[Value]) (Value, bool) {
	for v := range seq {
		return v, true
	}
	return Value{}, false
}

// parseAndEval is a test helper
func parseAndEval(t *testing.T, expr string, input Value) []Value {
	t.Helper()
	node, errs := Parse(expr)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	return collect(Eval(node, input, DefaultEnv()))
}

func testInput() Value {
	return ConcreteValue(map[string]any{
		"time":  "2024-01-15T10:30:00Z",
		"level": "INFO",
		"msg":   "server started",
		"port":  float64(8080),
		"host":  "localhost",
		"tags":  []any{"web", "prod"},
		"headers": map[string]any{
			"content-type": "application/json",
		},
	})
}

// Concrete evaluation tests

func TestEval_Identity(t *testing.T) {
	vals := parseAndEval(t, ".", testInput())
	if len(vals) != 1 {
		t.Fatalf("expected 1 result, got %d", len(vals))
	}
	if vals[0].Data == nil {
		t.Error("expected non-nil data")
	}
}

func TestEval_Field(t *testing.T) {
	vals := parseAndEval(t, ".level", testInput())
	if len(vals) != 1 || vals[0].Data != "INFO" {
		t.Errorf("expected INFO, got %v", vals)
	}
}

func TestEval_ChainedField(t *testing.T) {
	// .headers."content-type" uses string index via suffix
	// Test a simpler chain: .headers is accessed as a field
	vals := parseAndEval(t, `.headers`, testInput())
	if len(vals) != 1 {
		t.Fatalf("expected 1 result, got %d", len(vals))
	}
	if _, ok := vals[0].Data.(map[string]Value); !ok {
		t.Errorf("expected map[string]Value, got %T", vals[0].Data)
	}
}

func TestEval_Pipe(t *testing.T) {
	vals := parseAndEval(t, ".port | . + 1", testInput())
	if len(vals) != 1 {
		t.Fatalf("expected 1 result, got %d", len(vals))
	}
	if vals[0].Data != float64(8081) {
		t.Errorf("expected 8081, got %v", vals[0].Data)
	}
}

func TestEval_Select(t *testing.T) {
	vals := parseAndEval(t, `select(.level == "INFO")`, testInput())
	if len(vals) != 1 {
		t.Errorf("select should yield input when condition is true, got %d results", len(vals))
	}

	vals = parseAndEval(t, `select(.level == "ERROR")`, testInput())
	if len(vals) != 0 {
		t.Errorf("select should yield nothing when condition is false, got %d results", len(vals))
	}
}

func TestEval_Comparison(t *testing.T) {
	vals := parseAndEval(t, `.level == "INFO"`, testInput())
	if len(vals) != 1 || vals[0].Data != true {
		t.Errorf("expected true, got %v", vals)
	}

	vals = parseAndEval(t, ".port > 80", testInput())
	if len(vals) != 1 || vals[0].Data != true {
		t.Errorf("expected true for 8080 > 80, got %v", vals)
	}
}

func TestEval_Arithmetic(t *testing.T) {
	vals := parseAndEval(t, ".port + 1", testInput())
	if len(vals) != 1 || vals[0].Data != float64(8081) {
		t.Errorf("expected 8081, got %v", vals)
	}

	vals = parseAndEval(t, ".port - 80", testInput())
	if len(vals) != 1 || vals[0].Data != float64(8000) {
		t.Errorf("expected 8000, got %v", vals)
	}
}

func TestEval_StringConcat(t *testing.T) {
	vals := parseAndEval(t, `.level + "-" + .msg`, testInput())
	if len(vals) != 1 || vals[0].Data != "INFO-server started" {
		t.Errorf("expected 'INFO-server started', got %v", vals)
	}
}

func TestEval_Iterator(t *testing.T) {
	vals := parseAndEval(t, ".tags[]", testInput())
	if len(vals) != 2 {
		t.Fatalf("expected 2 results from .tags[], got %d", len(vals))
	}
	if vals[0].Data != "web" || vals[1].Data != "prod" {
		t.Errorf("expected [web, prod], got %v %v", vals[0].Data, vals[1].Data)
	}
}

func TestEval_ArrayIndex(t *testing.T) {
	vals := parseAndEval(t, ".tags[0]", testInput())
	if len(vals) != 1 || vals[0].Data != "web" {
		t.Errorf("expected web, got %v", vals)
	}
}

func TestEval_Length(t *testing.T) {
	vals := parseAndEval(t, ".tags | length", testInput())
	if len(vals) != 1 || vals[0].Data != float64(2) {
		t.Errorf("expected 2, got %v", vals)
	}
}

func TestEval_Keys(t *testing.T) {
	input := ConcreteValue(map[string]any{"b": float64(2), "a": float64(1)})
	vals := parseAndEval(t, "keys", input)
	if len(vals) != 1 {
		t.Fatalf("expected 1 result, got %d", len(vals))
	}
}

func TestEval_Map(t *testing.T) {
	input := ConcreteValue([]any{float64(1), float64(2), float64(3)})
	vals := parseAndEval(t, "map(. * 2)", input)
	if len(vals) != 1 {
		t.Fatalf("expected 1 result, got %d", len(vals))
	}
	arr, ok := vals[0].Data.([]Value)
	if !ok {
		t.Fatalf("expected []Value, got %T", vals[0].Data)
	}
	if len(arr) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr))
	}
	if arr[0].Data != float64(2) || arr[1].Data != float64(4) || arr[2].Data != float64(6) {
		t.Errorf("expected [2,4,6], got %v", arr)
	}
}

func TestEval_ToEntries(t *testing.T) {
	input := ConcreteValue(map[string]any{"a": float64(1), "b": float64(2)})
	vals := parseAndEval(t, "to_entries", input)
	if len(vals) != 1 {
		t.Fatalf("expected 1 result, got %d", len(vals))
	}
	arr, ok := vals[0].Data.([]Value)
	if !ok {
		t.Fatalf("expected []Value, got %T", vals[0].Data)
	}
	if len(arr) != 2 {
		t.Errorf("expected 2 entries, got %d", len(arr))
	}
}

func TestEval_IfThenElse(t *testing.T) {
	vals := parseAndEval(t, `if .level == "INFO" then "ok" else "bad" end`, testInput())
	if len(vals) != 1 || vals[0].Data != "ok" {
		t.Errorf("expected ok, got %v", vals)
	}
}

func TestEval_ObjectConstruction(t *testing.T) {
	vals := parseAndEval(t, `{level: .level, port: .port}`, testInput())
	if len(vals) != 1 {
		t.Fatalf("expected 1 result, got %d", len(vals))
	}
	data, ok := vals[0].Data.(map[string]Value)
	if !ok {
		t.Fatalf("expected map[string]Value, got %T", vals[0].Data)
	}
	if data["level"].Data != "INFO" {
		t.Errorf("level = %v, want INFO", data["level"].Data)
	}
}

func TestEval_ArrayConstruction(t *testing.T) {
	vals := parseAndEval(t, "[.level, .port]", testInput())
	if len(vals) != 1 {
		t.Fatalf("expected 1 result, got %d", len(vals))
	}
	arr, ok := vals[0].Data.([]Value)
	if !ok {
		t.Fatalf("expected []Value, got %T", vals[0].Data)
	}
	if len(arr) != 2 {
		t.Errorf("expected 2 elements, got %d", len(arr))
	}
}

func TestEval_Not(t *testing.T) {
	vals := parseAndEval(t, "true | not", testInput())
	if len(vals) != 1 || vals[0].Data != false {
		t.Errorf("expected false, got %v", vals)
	}
}

func TestEval_Alternative(t *testing.T) {
	vals := parseAndEval(t, ".missing // 42", testInput())
	if len(vals) != 1 || vals[0].Data != float64(42) {
		t.Errorf("expected 42, got %v", vals)
	}
}

func TestEval_Split(t *testing.T) {
	vals := parseAndEval(t, `.msg | split(" ")`, testInput())
	if len(vals) != 1 {
		t.Fatalf("expected 1 result, got %d", len(vals))
	}
}

func TestEval_Reduce(t *testing.T) {
	input := ConcreteValue([]any{float64(1), float64(2), float64(3)})
	vals := parseAndEval(t, "reduce .[] as $x (0; . + $x)", input)
	if len(vals) != 1 || vals[0].Data != float64(6) {
		t.Errorf("expected 6, got %v", vals)
	}
}

// Symbolic evaluation tests

func TestEvalSymbolic_Identity(t *testing.T) {
	input := SymbolicValue(ObjectShape{Fields: map[string]Shape{"x": StringShape{}}})
	vals := parseAndEval(t, ".", input)
	if len(vals) != 1 {
		t.Fatalf("expected 1, got %d", len(vals))
	}
	if _, ok := vals[0].Shape.(ObjectShape); !ok {
		t.Errorf("expected ObjectShape, got %T", vals[0].Shape)
	}
}

func TestEvalSymbolic_Field(t *testing.T) {
	input := SymbolicValue(ObjectShape{Fields: map[string]Shape{"level": StringShape{}}})
	vals := parseAndEval(t, ".level", input)
	if len(vals) != 1 {
		t.Fatalf("expected 1, got %d", len(vals))
	}
	if _, ok := vals[0].Shape.(StringShape); !ok {
		t.Errorf("expected StringShape, got %T", vals[0].Shape)
	}
	if !vals[0].Unknown {
		t.Error("expected Unknown to be true")
	}
}

func TestEvalSymbolic_Select(t *testing.T) {
	input := SymbolicValue(ObjectShape{Fields: map[string]Shape{"level": StringShape{}}})
	vals := parseAndEval(t, `select(.level == "ERROR")`, input)
	if len(vals) != 1 {
		t.Fatalf("expected 1, got %d", len(vals))
	}
	if _, ok := vals[0].Shape.(ObjectShape); !ok {
		t.Errorf("expected ObjectShape (preserved), got %T", vals[0].Shape)
	}
}

func TestEvalSymbolic_ToEntries(t *testing.T) {
	input := SymbolicValue(ObjectShape{Fields: map[string]Shape{
		"a": StringShape{}, "b": NumberShape{},
	}})
	vals := parseAndEval(t, "to_entries", input)
	if len(vals) != 1 {
		t.Fatalf("expected 1, got %d", len(vals))
	}
	arr, ok := vals[0].Shape.(ArrayShape)
	if !ok {
		t.Fatalf("expected ArrayShape, got %T", vals[0].Shape)
	}
	elem, ok := arr.Element.(ObjectShape)
	if !ok {
		t.Fatalf("expected ObjectShape element, got %T", arr.Element)
	}
	if _, ok := elem.Fields["key"].(StringShape); !ok {
		t.Error("expected key: StringShape")
	}
	if _, ok := elem.Fields["value"]; !ok {
		t.Error("expected value field")
	}
}

func TestEvalSymbolic_Keys(t *testing.T) {
	input := SymbolicValue(ObjectShape{Fields: map[string]Shape{"a": StringShape{}}})
	vals := parseAndEval(t, "keys", input)
	if len(vals) != 1 {
		t.Fatalf("expected 1, got %d", len(vals))
	}
	arr, ok := vals[0].Shape.(ArrayShape)
	if !ok {
		t.Fatalf("expected ArrayShape, got %T", vals[0].Shape)
	}
	if _, ok := arr.Element.(StringShape); !ok {
		t.Error("expected StringShape elements")
	}
}

func TestEvalSymbolic_Map(t *testing.T) {
	input := SymbolicValue(ArrayShape{Element: ObjectShape{Fields: map[string]Shape{"x": NumberShape{}}}})
	vals := parseAndEval(t, "map(.x)", input)
	if len(vals) != 1 {
		t.Fatalf("expected 1, got %d", len(vals))
	}
	arr, ok := vals[0].Shape.(ArrayShape)
	if !ok {
		t.Fatalf("expected ArrayShape, got %T", vals[0].Shape)
	}
	if _, ok := arr.Element.(NumberShape); !ok {
		t.Errorf("expected NumberShape element, got %T", arr.Element)
	}
}

func TestEvalSymbolic_Comparison(t *testing.T) {
	input := SymbolicValue(ObjectShape{Fields: map[string]Shape{"x": NumberShape{}}})
	vals := parseAndEval(t, `.x == 5`, input)
	if len(vals) != 1 {
		t.Fatalf("expected 1, got %d", len(vals))
	}
	if _, ok := vals[0].Shape.(BoolShape); !ok {
		t.Errorf("expected BoolShape, got %T", vals[0].Shape)
	}
}

func TestEvalSymbolic_IncompleteNode(t *testing.T) {
	// Parse an incomplete expression -- should still evaluate
	node, _ := Parse("select(.")
	input := SymbolicValue(ObjectShape{Fields: map[string]Shape{"x": NumberShape{}}})
	vals := collect(Eval(node, input, DefaultEnv()))
	if len(vals) != 1 {
		t.Fatalf("expected 1, got %d", len(vals))
	}
	// Should get something back (not crash)
}

func TestEvalSymbolic_UnknownField(t *testing.T) {
	input := SymbolicValue(AnyShape{})
	vals := parseAndEval(t, ".foo", input)
	if len(vals) != 1 {
		t.Fatalf("expected 1, got %d", len(vals))
	}
	if _, ok := vals[0].Shape.(AnyShape); !ok {
		t.Errorf("expected AnyShape for unknown field, got %T", vals[0].Shape)
	}
}
