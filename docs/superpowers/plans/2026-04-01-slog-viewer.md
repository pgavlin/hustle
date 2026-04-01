# Hustle: slog JSON Log Viewer — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a TUI log viewer that loads slog JSON log files and displays them in a tea-grid data grid with per-column filtering and a detail page.

**Architecture:** Nested bubbletea models. Top-level `Model` owns a `grid.Model[LogRecord]` and a `DetailModel`, routing input to whichever view is active. Data loads once at startup from a file path CLI argument.

**Tech Stack:** Go 1.25, bubbletea, lipgloss, github.com/pgavlin/tea-grid, bubbles (viewport if needed)

**API note:** The tea-grid API details below are based on exploration of the repository. If any constructor, option, or method signature doesn't match the actual API at implementation time, adjust the code to match the real API — the intent is clear even if exact names differ.

---

## File Structure

| File | Responsibility |
|------|---------------|
| `main.go` | CLI arg parsing, file loading, bubbletea program start |
| `log/record.go` | `LogRecord` type and single-line JSON parsing |
| `log/record_test.go` | Tests for JSON parsing |
| `log/loader.go` | Load all records from a file path |
| `log/loader_test.go` | Tests for file loading |
| `ui/detail.go` | `DetailModel` — structured/raw views, toggle, scroll |
| `ui/detail_test.go` | Tests for detail view rendering |
| `ui/grid.go` | Grid constructor — column defs, renderers, filters |
| `ui/app.go` | Top-level `Model` — view routing, key handling, status bar |

---

### Task 1: Set up Go module and dependencies

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add dependencies**

```bash
cd /Users/pgavlin/dev/sandbox/hustle
go get github.com/pgavlin/tea-grid@latest
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
```

- [ ] **Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add bubbletea and tea-grid dependencies"
```

---

### Task 2: LogRecord type and JSON parsing

**Files:**
- Create: `log/record.go`
- Create: `log/record_test.go`

- [ ] **Step 1: Write failing tests for ParseRecord**

```go
// log/record_test.go
package log

import (
	"testing"
	"time"
)

func TestParseRecord_ValidFullRecord(t *testing.T) {
	line := `{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"server started","port":8080,"host":"localhost"}`
	rec, err := ParseRecord(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want %q", rec.Level, "INFO")
	}
	if rec.Msg != "server started" {
		t.Errorf("msg = %q, want %q", rec.Msg, "server started")
	}
	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !rec.Time.Equal(expectedTime) {
		t.Errorf("time = %v, want %v", rec.Time, expectedTime)
	}
	if rec.Attrs["port"] != float64(8080) {
		t.Errorf("attrs[port] = %v, want %v", rec.Attrs["port"], float64(8080))
	}
	if rec.Attrs["host"] != "localhost" {
		t.Errorf("attrs[host] = %v, want %v", rec.Attrs["host"], "localhost")
	}
	if rec.RawJSON != line {
		t.Errorf("rawJSON mismatch")
	}
}

func TestParseRecord_MissingStandardFields(t *testing.T) {
	line := `{"extra":"value"}`
	rec, err := ParseRecord(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.Time.IsZero() {
		t.Errorf("time should be zero, got %v", rec.Time)
	}
	if rec.Level != "" {
		t.Errorf("level should be empty, got %q", rec.Level)
	}
	if rec.Msg != "" {
		t.Errorf("msg should be empty, got %q", rec.Msg)
	}
	if rec.Attrs["extra"] != "value" {
		t.Errorf("attrs[extra] = %v, want %q", rec.Attrs["extra"], "value")
	}
}

func TestParseRecord_NestedAttrs(t *testing.T) {
	line := `{"time":"2024-01-15T10:30:00Z","level":"DEBUG","msg":"request","headers":{"content-type":"application/json"}}`
	rec, err := ParseRecord(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	headers, ok := rec.Attrs["headers"].(map[string]any)
	if !ok {
		t.Fatalf("attrs[headers] is not a map, got %T", rec.Attrs["headers"])
	}
	if headers["content-type"] != "application/json" {
		t.Errorf("headers[content-type] = %v, want %q", headers["content-type"], "application/json")
	}
}

func TestParseRecord_MalformedJSON(t *testing.T) {
	_, err := ParseRecord("not json at all")
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestParseRecord_EmptyObject(t *testing.T) {
	rec, err := ParseRecord("{}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rec.Attrs) != 0 {
		t.Errorf("attrs should be empty, got %v", rec.Attrs)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./log/ -v`
Expected: compilation error — `ParseRecord` not defined.

- [ ] **Step 3: Implement LogRecord and ParseRecord**

```go
// log/record.go
package log

import (
	"encoding/json"
	"fmt"
	"time"
)

// LogRecord represents a single slog JSON log entry.
type LogRecord struct {
	Time    time.Time
	Level   string
	Msg     string
	Attrs   map[string]any
	RawJSON string
}

// ParseRecord parses a single JSON log line into a LogRecord.
func ParseRecord(line string) (LogRecord, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return LogRecord{}, fmt.Errorf("invalid JSON: %w", err)
	}

	rec := LogRecord{
		RawJSON: line,
		Attrs:   make(map[string]any),
	}

	if v, ok := raw["time"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			rec.Time = t
		}
	}
	if v, ok := raw["level"].(string); ok {
		rec.Level = v
	}
	if v, ok := raw["msg"].(string); ok {
		rec.Msg = v
	}

	for k, v := range raw {
		switch k {
		case "time", "level", "msg":
			continue
		default:
			rec.Attrs[k] = v
		}
	}

	return rec, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./log/ -v`
Expected: all 5 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add log/record.go log/record_test.go
git commit -m "feat: add LogRecord type and JSON parsing"
```

---

### Task 3: File loader

**Files:**
- Create: `log/loader.go`
- Create: `log/loader_test.go`

- [ ] **Step 1: Write failing tests for Load**

```go
// log/loader_test.go
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
	records, skipped, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0", skipped)
	}
	if records[0].Msg != "hello" {
		t.Errorf("records[0].Msg = %q, want %q", records[0].Msg, "hello")
	}
	if records[1].Attrs["code"] != float64(500) {
		t.Errorf("records[1].Attrs[code] = %v, want 500", records[1].Attrs["code"])
	}
}

func TestLoad_SkipsMalformedLines(t *testing.T) {
	path := writeTempFile(t, `{"level":"INFO","msg":"good"}
not json
{"level":"WARN","msg":"also good"}
`)
	records, skipped, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1", skipped)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	path := writeTempFile(t, "")
	records, skipped, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("got %d records, want 0", len(records))
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0", skipped)
	}
}

func TestLoad_SkipsBlankLines(t *testing.T) {
	path := writeTempFile(t, `{"level":"INFO","msg":"one"}

{"level":"INFO","msg":"two"}
`)
	records, skipped, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0", skipped)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, _, err := Load("/nonexistent/path/to/file.log")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./log/ -run TestLoad -v`
Expected: compilation error — `Load` not defined.

- [ ] **Step 3: Implement Load**

```go
// log/loader.go
package log

import (
	"bufio"
	"fmt"
	"os"
)

// Load reads a log file line-by-line, parsing each line as a JSON LogRecord.
// Returns the parsed records, count of skipped (malformed) lines, and any
// file-level error.
func Load(path string) ([]LogRecord, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var records []LogRecord
	skipped := 0
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		rec, err := ParseRecord(line)
		if err != nil {
			skipped++
			continue
		}
		records = append(records, rec)
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("read %s: %w", path, err)
	}

	return records, skipped, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./log/ -v`
Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add log/loader.go log/loader_test.go
git commit -m "feat: add file loader for slog JSON logs"
```

---

### Task 4: Detail model

**Files:**
- Create: `ui/detail.go`
- Create: `ui/detail_test.go`

- [ ] **Step 1: Write failing tests for detail view rendering**

```go
// ui/detail_test.go
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
	view := m.View()

	for _, want := range []string{"INFO", "server started", "port", "8080"} {
		if !strings.Contains(view, want) {
			t.Errorf("structured view missing %q", want)
		}
	}
}

func TestDetailModel_RawView(t *testing.T) {
	m := NewDetailModel(testRecord(), 80, 24)
	m.mode = detailRaw
	view := m.View()

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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./ui/ -run TestDetail -v`
Expected: compilation error — types not defined.

- [ ] **Step 3: Implement DetailModel**

```go
// ui/detail.go
package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	logpkg "github.com/pgavlin/hustle/log"
)

type detailMode int

const (
	detailStructured detailMode = iota
	detailRaw
)

var (
	detailLabelStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	detailValueStyle  = lipgloss.NewStyle()
	detailHeaderStyle = lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("57")).
				Padding(0, 1)
	detailHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// DetailModel displays a single LogRecord in structured or raw JSON format.
type DetailModel struct {
	record logpkg.LogRecord
	mode   detailMode
	width  int
	height int
	scroll int
}

// NewDetailModel creates a DetailModel for the given record.
func NewDetailModel(record logpkg.LogRecord, width, height int) DetailModel {
	return DetailModel{
		record: record,
		mode:   detailStructured,
		width:  width,
		height: height,
	}
}

func (m *DetailModel) toggleMode() {
	if m.mode == detailStructured {
		m.mode = detailRaw
	} else {
		m.mode = detailStructured
	}
	m.scroll = 0
}

// SetSize updates the viewport dimensions.
func (m *DetailModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles key messages for the detail view.
func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.toggleMode()
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			m.scroll++
		}
	}
	return m, nil
}

// View renders the detail view.
func (m DetailModel) View() string {
	var content string
	switch m.mode {
	case detailStructured:
		content = m.structuredView()
	case detailRaw:
		content = m.rawView()
	}

	header := detailHeaderStyle.Render(m.headerText())
	help := detailHelpStyle.Render("tab: toggle view • esc: back to grid • j/k: scroll")

	lines := strings.Split(content, "\n")
	if m.scroll >= len(lines) {
		m.scroll = max(len(lines)-1, 0)
	}
	visible := lines[m.scroll:]
	bodyHeight := m.height - 3
	if bodyHeight > 0 && len(visible) > bodyHeight {
		visible = visible[:bodyHeight]
	}

	return header + "\n\n" + strings.Join(visible, "\n") + "\n\n" + help
}

func (m DetailModel) headerText() string {
	modeLabel := "Structured"
	if m.mode == detailRaw {
		modeLabel = "Raw JSON"
	}
	return fmt.Sprintf(" Log Entry Detail — %s ", modeLabel)
}

func (m DetailModel) structuredView() string {
	var b strings.Builder

	writeField := func(label, value string) {
		b.WriteString(detailLabelStyle.Render(fmt.Sprintf("%12s", label)))
		b.WriteString("  ")
		b.WriteString(detailValueStyle.Render(value))
		b.WriteString("\n")
	}

	writeField("Time", m.record.Time.Format("2006-01-02 15:04:05.000 MST"))
	writeField("Level", m.record.Level)
	writeField("Message", m.record.Msg)

	if len(m.record.Attrs) > 0 {
		b.WriteString("\n")
		b.WriteString(detailLabelStyle.Render("  Attributes"))
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", 40))
		b.WriteString("\n")

		keys := make([]string, 0, len(m.record.Attrs))
		for k := range m.record.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			writeField(k, formatAttrValue(m.record.Attrs[k]))
		}
	}

	return b.String()
}

func (m DetailModel) rawView() string {
	var raw map[string]any
	if err := json.Unmarshal([]byte(m.record.RawJSON), &raw); err != nil {
		return m.record.RawJSON
	}
	pretty, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return m.record.RawJSON
	}
	return string(pretty)
}

func formatAttrValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case nil:
		return "null"
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./ui/ -run TestDetail -v`
Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add ui/detail.go ui/detail_test.go
git commit -m "feat: add detail view with structured and raw JSON modes"
```

---

### Task 5: Grid setup

**Files:**
- Create: `ui/grid.go`

- [ ] **Step 1: Implement grid column definitions and constructor**

```go
// ui/grid.go
package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/grid"
	"github.com/pgavlin/tea-grid/selection"

	logpkg "github.com/pgavlin/hustle/log"
)

func logColumns() []data.Column[logpkg.LogRecord] {
	return []data.Column[logpkg.LogRecord]{
		{
			ColumnID:   "time",
			HeaderName: "Time",
			ValueGetter: func(r logpkg.LogRecord) any {
				return r.Time
			},
			ValueFormatter: func(v any, r logpkg.LogRecord) string {
				return r.Time.Format("15:04:05.000")
			},
			Width:      14,
			Filterable: true,
			Filter:     filter.NewTimeFilter(),
			Sortable:   true,
		},
		{
			ColumnID:   "level",
			HeaderName: "Level",
			ValueGetter: func(r logpkg.LogRecord) any {
				return r.Level
			},
			Width:      7,
			Filterable: true,
			Filter:     filter.NewSetFilter([]any{"DEBUG", "INFO", "WARN", "ERROR"}),
			Sortable:   true,
		},
		{
			ColumnID:   "msg",
			HeaderName: "Message",
			ValueGetter: func(r logpkg.LogRecord) any {
				return r.Msg
			},
			Flex:       2,
			Filterable: true,
			Filter:     filter.NewTextFilter(),
			Sortable:   true,
		},
		{
			ColumnID:   "attrs",
			HeaderName: "Attributes",
			ValueGetter: func(r logpkg.LogRecord) any {
				return formatAttrs(r.Attrs)
			},
			Flex:       1,
			Filterable: true,
			Filter:     filter.NewTextFilter(),
		},
	}
}

func newLogGrid(records []logpkg.LogRecord, width, height int) grid.Model[logpkg.LogRecord] {
	return grid.New(
		grid.WithColumns(logColumns()),
		grid.WithRows(records),
		grid.WithRowID(func(r logpkg.LogRecord) string {
			return fmt.Sprintf("%d-%s-%s", r.Time.UnixNano(), r.Level, r.Msg)
		}),
		grid.WithWidth[logpkg.LogRecord](width),
		grid.WithHeight[logpkg.LogRecord](height),
		grid.WithSelection[logpkg.LogRecord](selection.SelectSingle),
		grid.WithQuickFilter[logpkg.LogRecord](true),
		grid.WithFocused[logpkg.LogRecord](true),
	)
}

func formatAttrs(attrs map[string]any) string {
	if len(attrs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, attrs[k]))
	}
	return strings.Join(parts, ", ")
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./ui/`

If there are tea-grid API mismatches (field names, option signatures, filter constructors), adjust the code to match the real API. The intent of each column definition is: time with time filter, level with set filter, msg with text filter, attrs with text filter.

Expected: successful compilation.

- [ ] **Step 3: Commit**

```bash
git add ui/grid.go
git commit -m "feat: add grid column definitions with filtering"
```

---

### Task 6: Top-level app model

**Files:**
- Create: `ui/app.go`

- [ ] **Step 1: Implement the top-level Model**

```go
// ui/app.go
package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pgavlin/tea-grid/grid"

	logpkg "github.com/pgavlin/hustle/log"
)

type appView int

const (
	viewGrid appView = iota
	viewDetail
)

var statusBarStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252")).
	Background(lipgloss.Color("235")).
	Padding(0, 1)

// Model is the top-level bubbletea model for hustle.
type Model struct {
	records []logpkg.LogRecord
	skipped int
	grid    grid.Model[logpkg.LogRecord]
	detail  DetailModel
	view    appView
	width   int
	height  int
}

// New creates the top-level model with loaded records.
func New(records []logpkg.LogRecord, skipped int) Model {
	return Model{
		records: records,
		skipped: skipped,
		view:    viewGrid,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		gridHeight := m.height - 1 // reserve 1 line for status bar
		m.grid = newLogGrid(m.records, m.width, gridHeight)
		m.detail.SetSize(m.width, m.height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.view == viewGrid {
				return m, tea.Quit
			}
		case "enter":
			if m.view == viewGrid {
				selected := m.grid.SelectedRows()
				if len(selected) > 0 {
					m.detail = NewDetailModel(selected[0], m.width, m.height)
					m.view = viewDetail
					return m, nil
				}
			}
		case "esc":
			if m.view == viewDetail {
				m.view = viewGrid
				return m, nil
			}
		}
	}

	switch m.view {
	case viewGrid:
		var cmd tea.Cmd
		m.grid, cmd = m.grid.Update(msg)
		return m, cmd
	case viewDetail:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	switch m.view {
	case viewDetail:
		return m.detail.View()
	default:
		return m.grid.View() + "\n" + m.statusBar()
	}
}

func (m Model) statusBar() string {
	text := fmt.Sprintf(" %d records", len(m.records))
	if m.skipped > 0 {
		text += fmt.Sprintf(", %d skipped", m.skipped)
	}
	return statusBarStyle.Width(m.width).Render(text)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./ui/`

If `grid.Model.Update` returns a different type (e.g. `tea.Model` instead of `grid.Model[T]`), or `SelectedRows()` has a different name, adjust accordingly. The intent: update the grid sub-model, get selected row on Enter, switch views.

Expected: successful compilation.

- [ ] **Step 3: Commit**

```bash
git add ui/app.go
git commit -m "feat: add top-level app model with grid/detail routing"
```

---

### Task 7: main.go and end-to-end wiring

**Files:**
- Create: `main.go`

- [ ] **Step 1: Implement main.go**

```go
// main.go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	logpkg "github.com/pgavlin/hustle/log"
	"github.com/pgavlin/hustle/ui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: hustle <logfile>\n")
		os.Exit(1)
	}

	path := os.Args[1]
	records, skipped, err := logpkg.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	m := ui.New(records, skipped)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Resolve dependencies**

```bash
cd /Users/pgavlin/dev/sandbox/hustle
go mod tidy
```

- [ ] **Step 3: Build the binary**

Run: `go build -o hustle .`
Expected: produces `hustle` binary with no errors.

- [ ] **Step 4: Create a test log file and run manually**

```bash
cat > /tmp/hustle-test.log <<'EOF'
{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"server started","port":8080}
{"time":"2024-01-15T10:30:01Z","level":"DEBUG","msg":"loading config","file":"config.yaml"}
{"time":"2024-01-15T10:30:02Z","level":"WARN","msg":"deprecated flag used","flag":"--old-mode"}
{"time":"2024-01-15T10:30:03Z","level":"ERROR","msg":"connection failed","host":"db.example.com","retries":3}
{"time":"2024-01-15T10:30:04Z","level":"INFO","msg":"request handled","method":"GET","path":"/api/health","duration_ms":12.5}
EOF
./hustle /tmp/hustle-test.log
```

Expected: interactive grid with 5 log entries. Verify:
- Columns: Time, Level, Message, Attributes
- Filtering works per column
- Enter opens detail view for selected row
- Tab toggles structured/raw in detail
- Esc returns to grid
- q or ctrl+c quits

- [ ] **Step 5: Run all tests**

Run: `go test ./... -v`
Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add main.go go.mod go.sum
git commit -m "feat: add main.go — hustle is runnable"
```
