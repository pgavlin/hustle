package ui

import (
	"strings"
	"testing"
	"time"

	logpkg "github.com/pgavlin/hustle/log"
)

func testRecord() logpkg.LogRecord {
	return logpkg.LogRecord{
		Time:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Level:   "INFO",
		Msg:     "server started",
		Attrs:   map[string]any{"port": float64(8080), "host": "localhost"},
		RawJSON: `{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"server started","port":8080,"host":"localhost"}`,
	}
}

func TestDetailModel_StructuredView(t *testing.T) {
	m := NewDetailModel(testRecord(), 80, 24)
	view := m.View().Content

	for _, want := range []string{"INFO", "server started", "port", "8080"} {
		if !strings.Contains(view, want) {
			t.Errorf("structured view missing %q", want)
		}
	}
}

func TestDetailModel_RawView(t *testing.T) {
	m := NewDetailModel(testRecord(), 80, 24)
	m.mode = detailRaw
	view := m.View().Content

	for _, want := range []string{`"level"`, `"server started"`} {
		if !strings.Contains(view, want) {
			t.Errorf("raw view missing %q", want)
		}
	}
}

func TestDetailModel_ToggleMode(t *testing.T) {
	m := NewDetailModel(testRecord(), 80, 24)
	if m.mode != detailStructured {
		t.Error("initial mode should be structured")
	}
	m.toggleMode()
	if m.mode != detailRaw {
		t.Error("after toggle, mode should be raw")
	}
	m.toggleMode()
	if m.mode != detailStructured {
		t.Error("after second toggle, mode should be structured")
	}
}
