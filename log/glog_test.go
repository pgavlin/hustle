package log

import (
	"testing"
	"time"
)

func TestGlog_Info(t *testing.T) {
	f := &GlogFormat{}
	rec, err := f.ParseRecord(`I0115 10:30:00.123456 12345 server.go:42] Server started on port 8080`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want INFO", rec.Level)
	}
	if rec.Msg != "Server started on port 8080" {
		t.Errorf("msg = %q", rec.Msg)
	}
	if rec.Time.Month() != time.January || rec.Time.Day() != 15 {
		t.Errorf("time = %v, want Jan 15", rec.Time)
	}
	if rec.Attrs["file"] != "server.go" {
		t.Errorf("file = %v", rec.Attrs["file"])
	}
	if rec.Attrs["line"] != float64(42) {
		t.Errorf("line = %v", rec.Attrs["line"])
	}
	if rec.Attrs["thread_id"] != float64(12345) {
		t.Errorf("thread_id = %v", rec.Attrs["thread_id"])
	}
}

func TestGlog_Warning(t *testing.T) {
	f := &GlogFormat{}
	rec, err := f.ParseRecord(`W0305 14:30:45.654321 99 config.go:87] Config file missing`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "WARNING" {
		t.Errorf("level = %q, want WARNING", rec.Level)
	}
}

func TestGlog_Error(t *testing.T) {
	f := &GlogFormat{}
	rec, err := f.ParseRecord(`E1110 08:15:22.000001 18765 database.go:156] Failed to connect`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "ERROR" {
		t.Errorf("level = %q, want ERROR", rec.Level)
	}
}

func TestGlog_Fatal(t *testing.T) {
	f := &GlogFormat{}
	rec, err := f.ParseRecord(`F1231 23:59:59.999999 1 main.go:10] Out of memory`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "FATAL" {
		t.Errorf("level = %q, want FATAL", rec.Level)
	}
}

func TestGlog_FullYear(t *testing.T) {
	f := &GlogFormat{}
	rec, err := f.ParseRecord(`I20260115 10:30:00.123456 12345 server.go:42] Hello`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Time.Year() != 2026 {
		t.Errorf("year = %d, want 2026", rec.Time.Year())
	}
}

func TestGlog_Invalid(t *testing.T) {
	f := &GlogFormat{}
	_, err := f.ParseRecord("not a glog line")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGlog_Name(t *testing.T) {
	f := &GlogFormat{}
	if f.Name() != "glog" {
		t.Errorf("name = %q, want glog", f.Name())
	}
}
