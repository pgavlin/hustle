package log

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata")
}

func loadTestWASM(t *testing.T, name string) *WASMFormat {
	t.Helper()
	wasmPath := filepath.Join(testdataDir(t), name)
	data, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Skipf("WASM test fixture not found: %v", err)
	}
	f, err := newWASMFormat(data)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestWASMFormat_GoPlugin(t *testing.T) {
	f := loadTestWASM(t, "example-go.wasm")

	if f.Name() != "example-go" {
		t.Errorf("name = %q, want example-go", f.Name())
	}

	rec, err := f.ParseRecord("INFO: server started port=8080")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want INFO", rec.Level)
	}
	if rec.Msg != "server started" {
		t.Errorf("msg = %q, want 'server started'", rec.Msg)
	}
	if rec.Attrs["port"] != "8080" {
		t.Errorf("port = %v (%T), want '8080'", rec.Attrs["port"], rec.Attrs["port"])
	}
	if rec.RawJSON != "INFO: server started port=8080" {
		t.Errorf("RawJSON = %q", rec.RawJSON)
	}
}

func TestWASMFormat_RustPlugin(t *testing.T) {
	f := loadTestWASM(t, "example-rust.wasm")

	if f.Name() != "example-rust" {
		t.Errorf("name = %q, want example-rust", f.Name())
	}

	rec, err := f.ParseRecord("ERROR: connection refused host=db.local retries=3")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "ERROR" {
		t.Errorf("level = %q, want ERROR", rec.Level)
	}
	if rec.Msg != "connection refused" {
		t.Errorf("msg = %q, want 'connection refused'", rec.Msg)
	}
	if rec.Attrs["host"] != "db.local" {
		t.Errorf("host = %v, want db.local", rec.Attrs["host"])
	}
	if rec.Attrs["retries"] != "3" {
		t.Errorf("retries = %v (%T), want '3'", rec.Attrs["retries"], rec.Attrs["retries"])
	}
}

func TestWASMFormat_ParseError(t *testing.T) {
	f := loadTestWASM(t, "example-go.wasm")

	_, err := f.ParseRecord("this is not a valid line")
	if err == nil {
		t.Error("expected error for unparseable line")
	}
}

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
