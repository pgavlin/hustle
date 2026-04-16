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
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# hello"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("some notes"), 0644)

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
	os.WriteFile(filepath.Join(dir, "bad.toml"), []byte("invalid"), 0644)

	original := append([]Format(nil), Formats...)
	defer func() { Formats = original }()

	warnings := LoadPluginsFrom(dir)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
}
