package filter

import (
	"testing"
	"time"

	logpkg "github.com/pgavlin/hustle/log"
)

func rec(level, msg string, attrs map[string]any) logpkg.LogRecord {
	var a logpkg.Attrs
	for k, v := range attrs {
		a.Set(k, v)
	}
	return logpkg.LogRecord{
		Time:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Level: level,
		Msg:   msg,
		Attrs: a,
	}
}

func TestCompile_ValidExpression(t *testing.T) {
	_, err := Compile(".level")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompile_InvalidSyntax(t *testing.T) {
	_, err := Compile(".foo | invalid!!")
	if err == nil {
		t.Error("expected error for invalid syntax")
	}
}

func TestMatch_SelectLevel(t *testing.T) {
	f, err := Compile(`select(.level == "ERROR")`)
	if err != nil {
		t.Fatal(err)
	}
	if !f.Match(rec("ERROR", "boom", nil)) {
		t.Error("expected ERROR record to match")
	}
	if f.Match(rec("INFO", "hello", nil)) {
		t.Error("expected INFO record not to match")
	}
}

func TestMatch_BooleanResult(t *testing.T) {
	f, err := Compile(`.level == "INFO"`)
	if err != nil {
		t.Fatal(err)
	}
	if !f.Match(rec("INFO", "hello", nil)) {
		t.Error("expected INFO record to match")
	}
	if f.Match(rec("ERROR", "boom", nil)) {
		t.Error("expected ERROR record not to match")
	}
}

func TestMatch_NestedAttrs(t *testing.T) {
	f, err := Compile(`.headers["content-type"] == "application/json"`)
	if err != nil {
		t.Fatal(err)
	}
	attrs := map[string]any{
		"headers": map[string]any{"content-type": "application/json"},
	}
	if !f.Match(rec("INFO", "req", attrs)) {
		t.Error("expected match on nested attr")
	}
}

func TestMatch_EmptyAttrs(t *testing.T) {
	f, err := Compile(`select(.level == "INFO")`)
	if err != nil {
		t.Fatal(err)
	}
	if !f.Match(rec("INFO", "hello", nil)) {
		t.Error("expected match with nil attrs")
	}
}

func TestMatch_NullOutput(t *testing.T) {
	f, err := Compile("null")
	if err != nil {
		t.Fatal(err)
	}
	if f.Match(rec("INFO", "hello", nil)) {
		t.Error("null output should not match")
	}
}

func TestMatch_FalseOutput(t *testing.T) {
	f, err := Compile("false")
	if err != nil {
		t.Fatal(err)
	}
	if f.Match(rec("INFO", "hello", nil)) {
		t.Error("false output should not match")
	}
}

func TestMatch_RuntimeError(t *testing.T) {
	f, err := Compile(".nonexistent | .foo")
	if err != nil {
		t.Fatal(err)
	}
	if f.Match(rec("INFO", "hello", nil)) {
		t.Error("runtime error should not match")
	}
}

func TestMatch_TimeField(t *testing.T) {
	f, err := Compile(`.time | startswith("2024")`)
	if err != nil {
		t.Fatal(err)
	}
	if !f.Match(rec("INFO", "hello", nil)) {
		t.Error("expected time field to be accessible")
	}
}

func TestMatch_MsgField(t *testing.T) {
	f, err := Compile(`.msg | test("hel")`)
	if err != nil {
		t.Fatal(err)
	}
	if !f.Match(rec("INFO", "hello", nil)) {
		t.Error("expected msg field to be accessible")
	}
}
