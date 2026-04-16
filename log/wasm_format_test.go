package log

import (
	"testing"
)

func TestWASMFormat_ParseResult(t *testing.T) {
	input := `{"ok":true,"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"hello","attrs":{"port":8080}}`
	rec, err := parseWASMResult([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want INFO", rec.Level)
	}
	if rec.Msg != "hello" {
		t.Errorf("msg = %q, want hello", rec.Msg)
	}
	if rec.Time.IsZero() {
		t.Error("time should not be zero")
	}
	if rec.Attrs["port"] != float64(8080) {
		t.Errorf("port = %v, want 8080", rec.Attrs["port"])
	}
}

func TestWASMFormat_ParseResultError(t *testing.T) {
	input := `{"ok":false,"error":"not recognized"}`
	_, err := parseWASMResult([]byte(input))
	if err == nil {
		t.Error("expected error for ok=false result")
	}
}

func TestWASMFormat_ParseResultInvalidJSON(t *testing.T) {
	_, err := parseWASMResult([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
