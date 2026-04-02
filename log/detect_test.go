package log

import (
	"strings"
	"testing"
)

func TestDetectFormat_JSON(t *testing.T) {
	input := `{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"hello"}
{"time":"2024-01-15T10:30:01Z","level":"ERROR","msg":"oops"}
`
	f, lines, err := DetectFormat(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if f.Name() != "json" {
		t.Errorf("detected %q, want json", f.Name())
	}
	if len(lines) != 2 {
		t.Errorf("sampled %d lines, want 2", len(lines))
	}
}

func TestDetectFormat_Logfmt(t *testing.T) {
	input := `time=2024-01-15T10:30:00Z level=info msg="hello" port=8080
time=2024-01-15T10:30:01Z level=error msg="oops"
`
	f, _, err := DetectFormat(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if f.Name() != "logfmt" {
		t.Errorf("detected %q, want logfmt", f.Name())
	}
}

func TestDetectFormat_CLF(t *testing.T) {
	input := `127.0.0.1 - - [15/Jan/2024:10:30:00 +0000] "GET / HTTP/1.1" 200 512 "-" "curl/7.0"
10.0.0.1 - - [15/Jan/2024:10:30:01 +0000] "POST /api HTTP/1.1" 201 128 "-" "curl/7.0"
`
	f, _, err := DetectFormat(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if f.Name() != "clf" {
		t.Errorf("detected %q, want clf", f.Name())
	}
}

func TestDetectFormat_Empty(t *testing.T) {
	f, _, err := DetectFormat(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if f.Name() != "json" {
		t.Errorf("empty input should default to json, got %q", f.Name())
	}
}

func TestDetectFormat_Unknown(t *testing.T) {
	input := `this is not a recognized format
neither is this line here
and this one is also unknown
`
	_, _, err := DetectFormat(strings.NewReader(input))
	if err == nil {
		t.Error("expected error for unrecognized format")
	}
}
