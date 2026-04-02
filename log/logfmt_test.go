package log

import "testing"

func TestLogfmt_Standard(t *testing.T) {
	f := &LogfmtFormat{}
	rec, err := f.ParseRecord(`time=2024-01-15T10:30:00Z level=info msg="server started" port=8080`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want INFO", rec.Level)
	}
	if rec.Msg != "server started" {
		t.Errorf("msg = %q, want 'server started'", rec.Msg)
	}
	if rec.Time.IsZero() {
		t.Error("time should not be zero")
	}
	if rec.Attrs["port"] != float64(8080) {
		t.Errorf("port = %v, want 8080", rec.Attrs["port"])
	}
}

func TestLogfmt_QuotedValues(t *testing.T) {
	f := &LogfmtFormat{}
	rec, err := f.ParseRecord(`level=error msg="something went wrong" path="/api/v1/users"`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Msg != "something went wrong" {
		t.Errorf("msg = %q", rec.Msg)
	}
	if rec.Attrs["path"] != "/api/v1/users" {
		t.Errorf("path = %v", rec.Attrs["path"])
	}
}

func TestLogfmt_BoolAndNumber(t *testing.T) {
	f := &LogfmtFormat{}
	rec, err := f.ParseRecord(`level=info msg=hello active=true count=42`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Attrs["active"] != true {
		t.Errorf("active = %v, want true", rec.Attrs["active"])
	}
	if rec.Attrs["count"] != float64(42) {
		t.Errorf("count = %v, want 42", rec.Attrs["count"])
	}
}

func TestLogfmt_MissingFields(t *testing.T) {
	f := &LogfmtFormat{}
	rec, err := f.ParseRecord(`key=value other=data`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "" {
		t.Errorf("level should be empty, got %q", rec.Level)
	}
	if rec.Attrs["key"] != "value" {
		t.Errorf("key = %v, want value", rec.Attrs["key"])
	}
}

func TestLogfmt_AlternateFieldNames(t *testing.T) {
	f := &LogfmtFormat{}
	rec, err := f.ParseRecord(`ts=2024-01-15T10:30:00Z severity=warn message="disk full"`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "WARN" {
		t.Errorf("level = %q, want WARN", rec.Level)
	}
	if rec.Msg != "disk full" {
		t.Errorf("msg = %q, want 'disk full'", rec.Msg)
	}
	if rec.Time.IsZero() {
		t.Error("time should not be zero")
	}
}

func TestLogfmt_Empty(t *testing.T) {
	f := &LogfmtFormat{}
	_, err := f.ParseRecord("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestLogfmt_Name(t *testing.T) {
	f := &LogfmtFormat{}
	if f.Name() != "logfmt" {
		t.Errorf("name = %q, want logfmt", f.Name())
	}
}
