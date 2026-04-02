package log

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_ValidFile(t *testing.T) {
	path := writeTempFile(t, `{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"hello"}
{"time":"2024-01-15T10:31:00Z","level":"ERROR","msg":"oops","code":500}
`)
	records, skipped, _, err := Load(path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0", skipped)
	}
	if records[0].Msg != "hello" {
		t.Errorf("records[0].Msg = %q, want %q", records[0].Msg, "hello")
	}
	if records[1].Attrs["code"] != float64(500) {
		t.Errorf("records[1].Attrs[code] = %v, want 500", records[1].Attrs["code"])
	}
}

func TestLoad_SkipsMalformedLines(t *testing.T) {
	path := writeTempFile(t, `{"level":"INFO","msg":"good"}
not json
{"level":"WARN","msg":"also good"}
`)
	records, skipped, _, err := Load(path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1", skipped)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	path := writeTempFile(t, "")
	records, skipped, _, err := Load(path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("got %d records, want 0", len(records))
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0", skipped)
	}
}

func TestLoad_SkipsBlankLines(t *testing.T) {
	path := writeTempFile(t, `{"level":"INFO","msg":"one"}

{"level":"INFO","msg":"two"}
`)
	records, skipped, _, err := Load(path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0", skipped)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, _, _, err := Load("/nonexistent/path/to/file.log", nil)
	if err == nil {
		t.Error("expected error for missing file")
	}
}
