package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
)

func DefaultPluginDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "hustle", "formats")
	}
	return ""
}

func LoadPlugins() {
	dir := DefaultPluginDir()
	if dir == "" {
		return
	}
	for _, w := range LoadPluginsFrom(dir) {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}
}

func LoadPluginsFrom(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []string{fmt.Sprintf("reading plugin dir %s: %v", dir, err)}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var warnings []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		ext := filepath.Ext(e.Name())

		var f Format
		var err error

		switch ext {
		case ".toml":
			f, err = loadRegexFormat(path)
		case ".wasm":
			f, err = loadWASMFormat(path)
		default:
			continue
		}

		if err != nil {
			warnings = append(warnings, fmt.Sprintf("loading %s: %v", e.Name(), err))
			continue
		}

		if existing := FormatByName(f.Name()); existing != nil {
			warnings = append(warnings, fmt.Sprintf(
				"plugin %s: name %q collides with existing format, skipping",
				e.Name(), f.Name()))
			continue
		}

		RegisterFormat(f)
	}

	return warnings
}

func loadRegexFormat(path string) (Format, error) {
	var cfg regexConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("parse TOML: %w", err)
	}
	return newRegexFormat(cfg)
}

func loadWASMFormat(path string) (Format, error) {
	return nil, fmt.Errorf("WASM format loading not yet implemented")
}
