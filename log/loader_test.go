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
	lf, err := Load(path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer lf.Close()
	if len(lf.Records) != 2 {
		t.Fatalf("got %d records, want 2", len(lf.Records))
	}
	if lf.Skipped != 0 {
		t.Errorf("skipped = %d, want 0", lf.Skipped)
	}
	if lf.Records[0].Msg != "hello" {
		t.Errorf("Records[0].Msg = %q, want %q", lf.Records[0].Msg, "hello")
	}
	if lf.Records[1].Attrs["code"] != float64(500) {
		t.Errorf("Records[1].Attrs[code] = %v, want 500", lf.Records[1].Attrs["code"])
	}
}

func TestLoad_SkipsMalformedLines(t *testing.T) {
	path := writeTempFile(t, `{"level":"INFO","msg":"good"}
not json
{"level":"WARN","msg":"also good"}
`)
	lf, err := Load(path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer lf.Close()
	if len(lf.Records) != 2 {
		t.Fatalf("got %d records, want 2", len(lf.Records))
	}
	if lf.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", lf.Skipped)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	path := writeTempFile(t, "")
	lf, err := Load(path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer lf.Close()
	if len(lf.Records) != 0 {
		t.Fatalf("got %d records, want 0", len(lf.Records))
	}
	if lf.Skipped != 0 {
		t.Errorf("skipped = %d, want 0", lf.Skipped)
	}
}

func TestLoad_SkipsBlankLines(t *testing.T) {
	path := writeTempFile(t, `{"level":"INFO","msg":"one"}

{"level":"INFO","msg":"two"}
`)
	lf, err := Load(path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer lf.Close()
	if len(lf.Records) != 2 {
		t.Fatalf("got %d records, want 2", len(lf.Records))
	}
	if lf.Skipped != 0 {
		t.Errorf("skipped = %d, want 0", lf.Skipped)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/to/file.log", nil)
	if err == nil {
		t.Error("expected error for missing file")
	}
}
