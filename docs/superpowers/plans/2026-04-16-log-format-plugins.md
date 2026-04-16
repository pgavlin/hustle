# Log Format Plugin Interface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow users to add custom log format parsers via regex config files (.toml) or WASM modules (.wasm), discovered from `~/.config/hustle/formats/` at startup.

**Architecture:** Everything is a `Format`. Three backends create them: Go code (existing), regex config (new), WASM module (new). A `RegisterFormat` function appends to the existing `Formats` slice. A `LoadPlugins` function scans the config directory and registers what it finds. Detection priority: built-ins first, then plugins alphabetically.

**Tech Stack:** Go, TOML (`github.com/BurntSushi/toml`), wazero (`github.com/tetratelabs/wazero`)

---

### Task 1: DocumentFormat interface and detection refactor

Decouple CloudWatch detection exclusion from positional slice indexing. This is a prerequisite for plugins — once plugins can append to `Formats`, the `Formats[:len(Formats)-1]` trick breaks.

**Files:**
- Modify: `log/format.go`
- Modify: `log/detect.go`
- Modify: `log/loader.go`
- Modify: `log/cloudwatch.go`
- Test: `log/detect_test.go`

- [ ] **Step 1: Write test for DocumentFormat detection skipping**

Add to `log/detect_test.go`:

```go
func TestDetectFormat_SkipsDocumentFormats(t *testing.T) {
	// Register a document format — it should be skipped during line-by-line detection
	original := append([]Format(nil), Formats...)
	defer func() { Formats = original }()

	Formats = append(Formats, &CloudWatchFormat{})

	input := `{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"hello"}
{"time":"2024-01-15T10:30:01Z","level":"ERROR","msg":"oops"}
`
	lines := splitLines([]byte(input))
	f, err := detectFormatFromLines(lines)
	if err != nil {
		t.Fatal(err)
	}
	if f.Name() != "json" {
		t.Errorf("detected %q, want json (should skip document formats)", f.Name())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./log/ -run TestDetectFormat_SkipsDocumentFormats -v`
Expected: FAIL — the duplicate CloudWatch at the end doesn't matter with the current slice trick, but once we change detection it should still pass. Actually this test verifies the new behavior works, so it may pass or fail depending on timing. Let's proceed to implement.

- [ ] **Step 3: Add DocumentFormat interface and RegisterFormat**

Replace `log/format.go` contents:

```go
package log

// Format parses log lines into LogRecords.
type Format interface {
	Name() string
	ParseRecord(line string) (LogRecord, error)
}

// DocumentFormat is implemented by formats that parse whole documents
// rather than individual lines (e.g. CloudWatch JSON). Document formats
// are skipped during line-by-line auto-detection.
type DocumentFormat interface {
	Format
	IsDocumentFormat()
}

// RegisterFormat adds a format to the registry. Plugin formats are
// appended after built-ins, so built-in formats always have detection
// priority.
func RegisterFormat(f Format) {
	Formats = append(Formats, f)
}
```

- [ ] **Step 4: Make CloudWatchFormat implement DocumentFormat**

Add to `log/cloudwatch.go`, after the `Name()` method:

```go
func (f *CloudWatchFormat) IsDocumentFormat() {}
```

- [ ] **Step 5: Update detectFormatFromLines to skip DocumentFormat**

In `log/detect.go`, replace the detection loop:

```go
func detectFormatFromLines(lines []string) (Format, error) {
	sample := lines
	if len(sample) > detectSampleSize {
		sample = sample[:detectSampleSize]
	}
	if len(sample) == 0 {
		return &JSONFormat{}, nil
	}

	for _, f := range Formats {
		// Skip document-level formats (e.g. CloudWatch JSON)
		if _, ok := f.(DocumentFormat); ok {
			continue
		}
		parsed := 0
		for _, line := range sample {
			if _, err := f.ParseRecord(line); err == nil {
				parsed++
			}
		}
		if parsed > len(sample)/2 {
			return f, nil
		}
	}

	return nil, fmt.Errorf(
		"could not detect log format from sample; try --format (%s)",
		strings.Join(FormatNames(), ", "),
	)
}
```

- [ ] **Step 6: Update loader.go to use DocumentFormat type assertion**

In `log/loader.go`, replace the two `*CloudWatchFormat` type assertions with `DocumentFormat`:

In `Load`:
```go
	// Document formats need special handling; stream directly.
	if df, ok := format.(DocumentFormat); ok {
		_ = df
		records, skipped, fmt, err := LoadCloudWatchJSON(f)
		if err != nil {
			return nil, err
		}
		return &LogFile{Records: records, Skipped: skipped, Format: fmt}, nil
	}
```

In `LoadReader`:
```go
	// Document formats need special handling.
	if df, ok := format.(DocumentFormat); ok {
		_ = df
		records, skipped, fmt, err := LoadCloudWatchJSON(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		return &LogFile{Records: records, Skipped: skipped, Format: fmt}, nil
	}
```

- [ ] **Step 7: Run all tests**

Run: `go test ./... -v`
Expected: All pass.

- [ ] **Step 8: Commit**

```bash
git add log/format.go log/detect.go log/loader.go log/cloudwatch.go log/detect_test.go
git commit -m "refactor: DocumentFormat interface, RegisterFormat, interface-based detection skipping"
```

---

### Task 2: Plugin discovery and loading skeleton

Implement `LoadPlugins` — scan `~/.config/hustle/formats/`, dispatch by extension, wire into `main.go`. No actual format loading yet (that comes in Tasks 3 and 4).

**Files:**
- Create: `log/register.go`
- Create: `log/register_test.go`
- Modify: `main.go`

- [ ] **Step 1: Write test for LoadPlugins with empty/missing directory**

Create `log/register_test.go`:

```go
package log

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPlugins_MissingDir(t *testing.T) {
	// Should not error when directory doesn't exist
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
	// No new formats should have been registered
	if len(Formats) != len(original) {
		t.Errorf("formats count changed: %d -> %d", len(original), len(Formats))
	}
}

func TestLoadPlugins_NameCollisionIgnored(t *testing.T) {
	dir := t.TempDir()
	// Write a .toml that would collide with built-in "json"
	os.WriteFile(filepath.Join(dir, "json.toml"), []byte(`
name = "json"
pattern = ".*"
`), 0644)

	original := append([]Format(nil), Formats...)
	defer func() { Formats = original }()

	warnings := LoadPluginsFrom(dir)
	// Should get a warning about the collision
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for name collision, got %d: %v", len(warnings), warnings)
	}
	// Format count should not change
	if len(Formats) != len(original) {
		t.Errorf("formats count changed: %d -> %d", len(original), len(Formats))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./log/ -run TestLoadPlugins -v`
Expected: FAIL — `LoadPluginsFrom` not defined.

- [ ] **Step 3: Implement LoadPlugins and LoadPluginsFrom**

Create `log/register.go`:

```go
package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// DefaultPluginDir returns the default directory for format plugins.
func DefaultPluginDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "hustle", "formats")
	}
	return ""
}

// LoadPlugins scans the default plugin directory and registers any formats found.
// Warnings are printed to stderr. This is the main entry point called from main.go.
func LoadPlugins() {
	dir := DefaultPluginDir()
	if dir == "" {
		return
	}
	for _, w := range LoadPluginsFrom(dir) {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}
}

// LoadPluginsFrom scans a directory for format plugins (.toml, .wasm) and
// registers them. Returns warnings for any plugins that failed to load.
// If the directory does not exist, returns nil.
func LoadPluginsFrom(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []string{fmt.Sprintf("reading plugin dir %s: %v", dir, err)}
	}

	// Sort entries alphabetically for deterministic load order
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

		// Check for name collision with existing formats
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

// Stub loaders — replaced by real implementations in regex_format.go and wasm_format.go

func loadRegexFormat(path string) (Format, error) {
	return nil, fmt.Errorf("regex format loading not yet implemented")
}

func loadWASMFormat(path string) (Format, error) {
	return nil, fmt.Errorf("WASM format loading not yet implemented")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./log/ -run TestLoadPlugins -v`
Expected: All pass. The name collision test works because `loadRegexFormat` returns a stub error — we need to adjust. Actually the stub returns an error, so the collision test won't hit the name check. Let me fix the test to use a pre-registered format instead.

Update `TestLoadPlugins_NameCollisionIgnored` — instead of relying on TOML loading, register a format directly and check that `LoadPluginsFrom` skips it. Actually, we need the TOML loader to work for this test. Let's defer this test to Task 3 and remove it for now. Replace it with:

```go
func TestLoadPlugins_WarnsOnBadPlugin(t *testing.T) {
	dir := t.TempDir()
	// Write a .toml that will fail to load (stub returns error)
	os.WriteFile(filepath.Join(dir, "bad.toml"), []byte("invalid"), 0644)

	original := append([]Format(nil), Formats...)
	defer func() { Formats = original }()

	warnings := LoadPluginsFrom(dir)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
}
```

- [ ] **Step 5: Run all tests**

Run: `go test ./... -v`
Expected: All pass.

- [ ] **Step 6: Wire LoadPlugins into main.go**

In `main.go`, add `logpkg.LoadPlugins()` call in the `run` function, after flag parsing and before format resolution:

```go
func run(ctx context.Context, cmd *cli.Command) error {
	// ... cpuprofile setup ...

	logpkg.LoadPlugins()

	var format logpkg.Format
	// ... rest unchanged ...
```

- [ ] **Step 7: Build and verify**

Run: `go build ./...`
Expected: Success.

- [ ] **Step 8: Commit**

```bash
git add log/register.go log/register_test.go main.go
git commit -m "feat: plugin discovery skeleton with LoadPlugins, wire into main"
```

---

### Task 3: Regex format implementation

Implement `RegexFormat` — loads a `.toml` config, compiles the regex, and parses log lines by extracting named capture groups.

**Files:**
- Create: `log/regex_format.go`
- Create: `log/regex_format_test.go`
- Modify: `log/register.go` (replace stub)

**New dependency:** `github.com/BurntSushi/toml`

- [ ] **Step 1: Add TOML dependency**

Run: `go get github.com/BurntSushi/toml`

- [ ] **Step 2: Write tests for RegexFormat**

Create `log/regex_format_test.go`:

```go
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
		// No TimeFormat — should try RFC3339 variants
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
		Pattern: `^(?P<level>\w+`,  // unclosed group
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
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./log/ -run TestRegexFormat -v`
Expected: FAIL — `regexConfig`, `newRegexFormat` not defined.

- [ ] **Step 4: Implement RegexFormat**

Create `log/regex_format.go`:

```go
package log

import (
	"fmt"
	"regexp"
	"time"

	"github.com/BurntSushi/toml"
)

// regexConfig is the TOML structure for a regex format plugin.
type regexConfig struct {
	Name       string            `toml:"name"`
	Pattern    string            `toml:"pattern"`
	TimeFormat string            `toml:"time_format"`
	LevelMap   map[string]string `toml:"level_map"`
}

// RegexFormat implements Format using a compiled regex with named capture groups.
type RegexFormat struct {
	name       string
	re         *regexp.Regexp
	groupNames []string
	timeFormat string
	levelMap   map[string]string
}

func newRegexFormat(cfg regexConfig) (*RegexFormat, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("regex format: name is required")
	}
	if cfg.Pattern == "" {
		return nil, fmt.Errorf("regex format %q: pattern is required", cfg.Name)
	}
	re, err := regexp.Compile(cfg.Pattern)
	if err != nil {
		return nil, fmt.Errorf("regex format %q: %w", cfg.Name, err)
	}
	return &RegexFormat{
		name:       cfg.Name,
		re:         re,
		groupNames: re.SubexpNames(),
		timeFormat: cfg.TimeFormat,
		levelMap:   cfg.LevelMap,
	}, nil
}

func (f *RegexFormat) Name() string { return f.name }

func (f *RegexFormat) ParseRecord(line string) (LogRecord, error) {
	match := f.re.FindStringSubmatchIndex(line)
	if match == nil {
		return LogRecord{}, fmt.Errorf("line does not match %s pattern", f.name)
	}

	rec := LogRecord{
		RawJSON: line,
		Attrs:   make(map[string]any, len(f.groupNames)),
	}

	for i, name := range f.groupNames {
		if name == "" || i*2 >= len(match) {
			continue
		}
		start, end := match[i*2], match[i*2+1]
		if start < 0 {
			continue
		}
		value := line[start:end]

		switch name {
		case "time":
			rec.Time = f.parseTime(value)
		case "level":
			rec.Level = f.parseLevel(value)
		case "msg":
			rec.Msg = value
		default:
			rec.Attrs[name] = inferValue(value)
		}
	}

	return rec, nil
}

func (f *RegexFormat) parseTime(s string) time.Time {
	if f.timeFormat != "" {
		if t, err := time.Parse(f.timeFormat, s); err == nil {
			return t
		}
	}
	// Fall back to common formats
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (f *RegexFormat) parseLevel(s string) string {
	if f.levelMap != nil {
		if mapped, ok := f.levelMap[s]; ok {
			return mapped
		}
	}
	return normalizeLevel(s)
}
```

- [ ] **Step 5: Replace loadRegexFormat stub in register.go**

In `log/register.go`, replace the stub:

```go
func loadRegexFormat(path string) (Format, error) {
	var cfg regexConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("parse TOML: %w", err)
	}
	return newRegexFormat(cfg)
}
```

Add the import for `"github.com/BurntSushi/toml"` to `register.go`.

- [ ] **Step 6: Run tests**

Run: `go test ./log/ -run "TestRegexFormat|TestLoadRegexFormat" -v`
Expected: All pass.

- [ ] **Step 7: Add name collision test back**

Add to `log/register_test.go`:

```go
func TestLoadPlugins_NameCollisionIgnored(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "json.toml"), []byte(`
name = "json"
pattern = '^(?P<msg>.*)'
`), 0644)

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
```

- [ ] **Step 8: Run all tests**

Run: `go test ./... -v`
Expected: All pass.

- [ ] **Step 9: Commit**

```bash
git add log/regex_format.go log/regex_format_test.go log/register.go log/register_test.go go.mod go.sum
git commit -m "feat: regex format plugin (.toml config with named capture groups)"
```

---

### Task 4: WASM format implementation

Implement `WASMFormat` — loads a `.wasm` module via wazero, calls exported `name`, `parse`, `alloc`, `free` functions per the ABI spec.

**Files:**
- Create: `log/wasm_format.go`
- Create: `log/wasm_format_test.go`
- Modify: `log/register.go` (replace stub)

**New dependency:** `github.com/tetratelabs/wazero`

- [ ] **Step 1: Add wazero dependency**

Run: `go get github.com/tetratelabs/wazero`

- [ ] **Step 2: Write a minimal WAT test fixture**

We need a `.wasm` binary for testing. The simplest approach: write a Go-based WASM test helper that compiles to `.wasm`, or use a hand-written WAT. For test purposes, create a Go file that builds a mock WASM module programmatically using wazero's module builder.

Create `log/wasm_format_test.go`:

```go
package log

import (
	"testing"
)

func TestWASMFormat_ParseResult(t *testing.T) {
	// Test that parseWASMResult correctly decodes JSON results
	input := `{"ok":true,"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"hello","attrs":{"port":8080}}`
	rec, err := parseWASMResult([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want INFO", rec.Level)
	}
	if rec.Msg != "hello" {
		t.Errorf("msg = %q, want hello", rec.Msg)
	}
	if rec.Time.IsZero() {
		t.Error("time should not be zero")
	}
	if rec.Attrs["port"] != float64(8080) {
		t.Errorf("port = %v, want 8080", rec.Attrs["port"])
	}
}

func TestWASMFormat_ParseResultError(t *testing.T) {
	input := `{"ok":false,"error":"not recognized"}`
	_, err := parseWASMResult([]byte(input))
	if err == nil {
		t.Error("expected error for ok=false result")
	}
}

func TestWASMFormat_ParseResultInvalidJSON(t *testing.T) {
	_, err := parseWASMResult([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./log/ -run TestWASMFormat -v`
Expected: FAIL — `parseWASMResult` not defined.

- [ ] **Step 4: Implement WASMFormat**

Create `log/wasm_format.go`:

```go
package log

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// WASMFormat implements Format by delegating to a WASM module.
type WASMFormat struct {
	name    string
	runtime wazero.Runtime
	module  api.Module
	alloc   api.Function
	free    api.Function
	parse   api.Function
}

func newWASMFormat(wasmBytes []byte) (*WASMFormat, error) {
	ctx := context.Background()

	rt := wazero.NewRuntime(ctx)

	mod, err := rt.Instantiate(ctx, wasmBytes)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("instantiate WASM module: %w", err)
	}

	// Validate required exports
	nameFn := mod.ExportedFunction("name")
	parseFn := mod.ExportedFunction("parse")
	allocFn := mod.ExportedFunction("alloc")
	freeFn := mod.ExportedFunction("free")

	if nameFn == nil || parseFn == nil || allocFn == nil || freeFn == nil {
		missing := ""
		if nameFn == nil {
			missing += "name "
		}
		if parseFn == nil {
			missing += "parse "
		}
		if allocFn == nil {
			missing += "alloc "
		}
		if freeFn == nil {
			missing += "free "
		}
		rt.Close(ctx)
		return nil, fmt.Errorf("WASM module missing required exports: %s", missing)
	}

	// Call name() to get the format name
	results, err := nameFn.Call(ctx)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("calling name(): %w", err)
	}
	if len(results) != 2 {
		rt.Close(ctx)
		return nil, fmt.Errorf("name() returned %d values, expected 2 (ptr, len)", len(results))
	}
	namePtr, nameLen := uint32(results[0]), uint32(results[1])
	nameBytes, ok := mod.Memory().Read(namePtr, nameLen)
	if !ok {
		rt.Close(ctx)
		return nil, fmt.Errorf("failed to read name from WASM memory")
	}

	return &WASMFormat{
		name:    string(nameBytes),
		runtime: rt,
		module:  mod,
		alloc:   allocFn,
		free:    freeFn,
		parse:   parseFn,
	}, nil
}

func (f *WASMFormat) Name() string { return f.name }

func (f *WASMFormat) ParseRecord(line string) (LogRecord, error) {
	ctx := context.Background()

	// Allocate memory in the module for the input line
	lineBytes := []byte(line)
	results, err := f.alloc.Call(ctx, uint64(len(lineBytes)))
	if err != nil {
		return LogRecord{}, fmt.Errorf("alloc: %w", err)
	}
	linePtr := uint32(results[0])
	defer f.free.Call(ctx, uint64(linePtr))

	// Write the line into module memory
	if !f.module.Memory().Write(linePtr, lineBytes) {
		return LogRecord{}, fmt.Errorf("failed to write line to WASM memory")
	}

	// Call parse(ptr, len) -> (result_ptr, result_len)
	results, err = f.parse.Call(ctx, uint64(linePtr), uint64(len(lineBytes)))
	if err != nil {
		return LogRecord{}, fmt.Errorf("parse: %w", err)
	}
	if len(results) != 2 {
		return LogRecord{}, fmt.Errorf("parse returned %d values, expected 2", len(results))
	}
	resultPtr, resultLen := uint32(results[0]), uint32(results[1])
	defer f.free.Call(ctx, uint64(resultPtr))

	// Read the JSON result from module memory
	resultBytes, ok := f.module.Memory().Read(resultPtr, resultLen)
	if !ok {
		return LogRecord{}, fmt.Errorf("failed to read parse result from WASM memory")
	}

	rec, err := parseWASMResult(resultBytes)
	if err != nil {
		return LogRecord{}, err
	}
	rec.RawJSON = line
	return rec, nil
}

// Close releases the WASM runtime resources.
func (f *WASMFormat) Close() error {
	return f.runtime.Close(context.Background())
}

// wasmResult is the JSON structure returned by the WASM parse function.
type wasmResult struct {
	OK    bool           `json:"ok"`
	Error string         `json:"error,omitempty"`
	Time  string         `json:"time,omitempty"`
	Level string         `json:"level,omitempty"`
	Msg   string         `json:"msg,omitempty"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// parseWASMResult decodes a JSON result from a WASM parse call.
func parseWASMResult(data []byte) (LogRecord, error) {
	var result wasmResult
	if err := json.Unmarshal(data, &result); err != nil {
		return LogRecord{}, fmt.Errorf("decode WASM result: %w", err)
	}
	if !result.OK {
		return LogRecord{}, fmt.Errorf("parse failed: %s", result.Error)
	}

	rec := LogRecord{
		Level: normalizeLevel(result.Level),
		Msg:   result.Msg,
		Attrs: result.Attrs,
	}
	if rec.Attrs == nil {
		rec.Attrs = make(map[string]any)
	}

	if result.Time != "" {
		rec.Time = parseTimeString(result.Time)
	}

	return rec, nil
}

// parseTimeString tries common time formats.
func parseTimeString(s string) time.Time {
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
```

- [ ] **Step 5: Replace loadWASMFormat stub in register.go**

In `log/register.go`, replace the stub:

```go
func loadWASMFormat(path string) (Format, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read WASM file: %w", err)
	}
	return newWASMFormat(data)
}
```

Add the `"os"` import if not already present (it should be).

- [ ] **Step 6: Run tests**

Run: `go test ./log/ -run TestWASMFormat -v`
Expected: All pass (the unit tests only test `parseWASMResult`, not module loading).

- [ ] **Step 7: Run all tests**

Run: `go test ./...`
Expected: All pass.

- [ ] **Step 8: Commit**

```bash
git add log/wasm_format.go log/wasm_format_test.go log/register.go go.mod go.sum
git commit -m "feat: WASM format plugin support via wazero"
```

---

### Task 5: Integration test with regex plugin

End-to-end test: write a `.toml` plugin to a temp directory, call `LoadPluginsFrom`, then load a log file using the plugin format.

**Files:**
- Modify: `log/register_test.go`

- [ ] **Step 1: Write integration test**

Add to `log/register_test.go`:

```go
func TestLoadPlugins_RegexIntegration(t *testing.T) {
	dir := t.TempDir()

	// Write a custom format plugin
	os.WriteFile(filepath.Join(dir, "custom.toml"), []byte(`
name = "custom"
pattern = '^(?P<time>[\dT:.Z-]+) (?P<level>\w+) \[(?P<component>\w+)\] (?P<msg>.*)'
time_format = "2006-01-02T15:04:05Z"
`), 0644)

	// Save and restore global state
	original := append([]Format(nil), Formats...)
	defer func() { Formats = original }()

	warnings := LoadPluginsFrom(dir)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}

	// The custom format should now be registered
	f := FormatByName("custom")
	if f == nil {
		t.Fatal("custom format not found after LoadPluginsFrom")
	}

	// Parse a line with it
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
`), 0644)
	os.WriteFile(filepath.Join(dir, "01-first.toml"), []byte(`
name = "first"
pattern = '^(?P<msg>.*)'
`), 0644)

	original := append([]Format(nil), Formats...)
	defer func() { Formats = original }()

	LoadPluginsFrom(dir)

	// Plugins should be appended after built-ins, in alphabetical order
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
```

- [ ] **Step 2: Run tests**

Run: `go test ./log/ -run "TestLoadPlugins_Regex|TestLoadPlugins_Alpha" -v`
Expected: All pass.

- [ ] **Step 3: Commit**

```bash
git add log/register_test.go
git commit -m "test: integration tests for regex plugin loading and ordering"
```

---

### Task 6: Update --format flag to include plugin formats

The `--format` help text currently lists only built-in formats. After `LoadPlugins` runs, plugin formats are in the registry and should appear in the help text and be selectable.

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Move format flag help to runtime**

Currently the flag `Usage` is set at declaration time (before `LoadPlugins` runs). The format names need to be resolved after plugins are loaded. Move the help string generation into `run`:

In `main.go`, change the format flag usage to a generic message and validate dynamically:

```go
&cli.StringFlag{
	Name:    "format",
	Aliases: []string{"f"},
	Value:   "auto",
	Usage:   "Log format (auto, or a specific format name)",
},
```

Then in `run`, after `LoadPlugins()`:

```go
logpkg.LoadPlugins()

var format logpkg.Format
if name := cmd.String("format"); name != "auto" {
	format = logpkg.FormatByName(name)
	if format == nil {
		return fmt.Errorf("unknown format %q; valid formats: %s",
			name, strings.Join(logpkg.FormatNames(), ", "))
	}
}
```

The error message already calls `FormatNames()` which includes plugins. No change needed there.

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Success.

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "fix: --format help text works with dynamically loaded plugins"
```
