package log

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRegexFormat_BasicParse(t *testing.T) {
	cfg := regexConfig{
		Name:       "myapp",
		Pattern:    `^(?P<time>\d{4}-\d{2}-\d{2}T[\d:.]+Z)\s+\[(?P<level>\w+)\]\s+(?P<msg>.*)`,
		TimeFormat: "2006-01-02T15:04:05.000Z",
	}
	f, err := newRegexFormat(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if f.Name() != "myapp" {
		t.Errorf("name = %q, want myapp", f.Name())
	}
	rec, err := f.ParseRecord("2024-01-15T10:30:00.000Z [INFO] server started")
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
	if rec.RawJSON != "2024-01-15T10:30:00.000Z [INFO] server started" {
		t.Errorf("RawJSON = %q", rec.RawJSON)
	}
}

func TestRegexFormat_LevelMap(t *testing.T) {
	cfg := regexConfig{
		Name:    "mapped",
		Pattern: `^(?P<level>\w+): (?P<msg>.*)`,
		LevelMap: map[string]string{
			"dbg": "DEBUG",
			"inf": "INFO",
			"wrn": "WARN",
			"err": "ERROR",
		},
	}
	f, err := newRegexFormat(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := f.ParseRecord("inf: hello world")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want INFO", rec.Level)
	}
}

func TestRegexFormat_ExtraGroupsAsAttrs(t *testing.T) {
	cfg := regexConfig{
		Name:    "extras",
		Pattern: `^(?P<level>\w+) (?P<pid>\d+) (?P<msg>.*)`,
	}
	f, err := newRegexFormat(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := f.ParseRecord("INFO 12345 request handled")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Attrs["pid"] != float64(12345) {
		t.Errorf("pid = %v, want 12345", rec.Attrs["pid"])
	}
}

func TestRegexFormat_NoMatch(t *testing.T) {
	cfg := regexConfig{
		Name:    "strict",
		Pattern: `^(?P<level>INFO|ERROR) (?P<msg>.*)`,
	}
	f, err := newRegexFormat(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.ParseRecord("this does not match")
	if err == nil {
		t.Error("expected error for non-matching line")
	}
}

func TestRegexFormat_DefaultTimeFormats(t *testing.T) {
	cfg := regexConfig{
		Name:    "autotime",
		Pattern: `^(?P<time>[\dT:.Z+-]+) (?P<msg>.*)`,
	}
	f, err := newRegexFormat(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := f.ParseRecord("2024-01-15T10:30:00Z hello")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Time.Year() != 2024 || rec.Time.Month() != time.January {
		t.Errorf("time = %v", rec.Time)
	}
}

func TestRegexFormat_BadPattern(t *testing.T) {
	cfg := regexConfig{
		Name:    "bad",
		Pattern: `^(?P<level>\w+`,
	}
	_, err := newRegexFormat(cfg)
	if err == nil {
		t.Error("expected error for bad regex")
	}
}

func TestLoadRegexFormat_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "myapp.toml")
	os.WriteFile(path, []byte(`
name = "myapp"
pattern = '^(?P<level>\w+) (?P<msg>.*)'
`), 0644)
	f, err := loadRegexFormat(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Name() != "myapp" {
		t.Errorf("name = %q, want myapp", f.Name())
	}
}

func TestLoadRegexFormat_MissingName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noname.toml")
	os.WriteFile(path, []byte(`
pattern = '^(?P<level>\w+) (?P<msg>.*)'
`), 0644)
	_, err := loadRegexFormat(path)
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestLoadRegexFormat_MissingPattern(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nopat.toml")
	os.WriteFile(path, []byte(`
name = "nopat"
`), 0644)
	_, err := loadRegexFormat(path)
	if err == nil {
		t.Error("expected error for missing pattern")
	}
}
