package log

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPlugins_MissingDir(t *testing.T) {
	warnings := LoadPluginsFrom("/nonexistent/path/to/formats")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for missing dir, got %v", warnings)
	}
}

func TestLoadPlugins_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	warnings := LoadPluginsFrom(dir)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for empty dir, got %v", warnings)
	}
}

func TestLoadPlugins_IgnoresUnknownExtensions(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# hello"), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("some notes"), 0o644)

	original := append([]Format(nil), Formats...)
	defer func() { Formats = original }()

	warnings := LoadPluginsFrom(dir)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(Formats) != len(original) {
		t.Errorf("formats count changed: %d -> %d", len(original), len(Formats))
	}
}

func TestLoadPlugins_WarnsOnBadPlugin(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.toml"), []byte("invalid"), 0o644)

	original := append([]Format(nil), Formats...)
	defer func() { Formats = original }()

	warnings := LoadPluginsFrom(dir)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
}

func TestLoadPlugins_RegexIntegration(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "custom.toml"), []byte(`
name = "custom"
pattern = '^(?P<time>[\dT:.Z-]+) (?P<level>\w+) \[(?P<component>\w+)\] (?P<msg>.*)'
time_format = "2006-01-02T15:04:05Z"
`), 0o644)

	original := append([]Format(nil), Formats...)
	defer func() { Formats = original }()

	warnings := LoadPluginsFrom(dir)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}

	f := FormatByName("custom")
	if f == nil {
		t.Fatal("custom format not found after LoadPluginsFrom")
	}

	rec, err := f.ParseRecord("2024-01-15T10:30:00Z INFO [auth] user logged in")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want INFO", rec.Level)
	}
	if rec.Msg != "user logged in" {
		t.Errorf("msg = %q", rec.Msg)
	}
	if rec.Attrs["component"] != "auth" {
		t.Errorf("component = %v, want auth", rec.Attrs["component"])
	}
	if rec.Time.Year() != 2024 {
		t.Errorf("time = %v", rec.Time)
	}
}

func TestLoadPlugins_AlphabeticalOrder(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "02-second.toml"), []byte(`
name = "second"
pattern = '^(?P<msg>.*)'
`), 0o644)
	os.WriteFile(filepath.Join(dir, "01-first.toml"), []byte(`
name = "first"
pattern = '^(?P<msg>.*)'
`), 0o644)

	original := append([]Format(nil), Formats...)
	defer func() { Formats = original }()

	LoadPluginsFrom(dir)

	pluginStart := len(original)
	if len(Formats) != len(original)+2 {
		t.Fatalf("expected %d formats, got %d", len(original)+2, len(Formats))
	}
	if Formats[pluginStart].Name() != "first" {
		t.Errorf("first plugin = %q, want 'first'", Formats[pluginStart].Name())
	}
	if Formats[pluginStart+1].Name() != "second" {
		t.Errorf("second plugin = %q, want 'second'", Formats[pluginStart+1].Name())
	}
}

func TestLoadPlugins_NameCollisionIgnored(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "json.toml"), []byte(`
name = "json"
pattern = '^(?P<msg>.*)'
`), 0o644)

	original := append([]Format(nil), Formats...)
	defer func() { Formats = original }()

	warnings := LoadPluginsFrom(dir)
	found := false
	for _, w := range warnings {
		if len(w) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for name collision with built-in")
	}
	if len(Formats) != len(original) {
		t.Errorf("formats count changed: %d -> %d", len(original), len(Formats))
	}
}
