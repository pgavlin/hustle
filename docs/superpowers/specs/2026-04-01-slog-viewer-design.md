# Hustle: slog JSON Log Viewer

## Overview

Hustle is a terminal application for viewing structured logs produced by Go's
`log/slog` package with the JSON handler. It reads a log file, displays entries
in a tea-grid data grid, and provides a detail page for inspecting individual
records.

## Input

- Single CLI argument: a file path.
- The file is read once at startup (no tailing / live streaming).
- Each line is parsed as a JSON object. Malformed lines are skipped.

## Data Model

```go
type LogRecord struct {
    Time    time.Time
    Level   string
    Msg     string
    Attrs   map[string]any // all fields beyond time, level, msg
    RawJSON string         // original line for the raw detail view
}
```

Parsing extracts the three standard slog fields (`time`, `level`, `msg`) and
collects everything else into `Attrs`. The original JSON line is preserved in
`RawJSON`.

## Architecture

**Approach**: Nested bubbletea models. A top-level `Model` owns the grid and
detail sub-models and routes input to whichever is active.

### Package Layout

```
hustle/
  main.go              — CLI arg parsing, file loading, bubbletea program start
  log/
    record.go          — LogRecord type and JSON parsing
    loader.go          — load all records from a file (line-by-line)
  ui/
    app.go             — top-level Model (view enum, input routing)
    grid.go            — grid.Model[LogRecord] setup: columns, renderers, filters
    detail.go          — DetailModel: structured/raw views for one record
```

### Top-Level Model (`ui/app.go`)

Fields:
- `records []log.LogRecord` — all parsed records
- `grid` — wraps `grid.Model[LogRecord]`
- `detail` — `DetailModel` for the detail page
- `view` — enum: `viewGrid` | `viewDetail`
- `skipped int` — count of malformed lines (shown in status bar)

Behavior:
- Routes `tea.KeyMsg` and `tea.WindowSizeMsg` to the active sub-model.
- Enter on a grid row transitions to `viewDetail` with the selected record.
- Esc from detail returns to `viewGrid`, preserving grid scroll and selection.

### Grid View (`ui/grid.go`)

Columns:
| Column | Source         | Filter                |
|--------|----------------|-----------------------|
| Time   | `Time`         | tea-grid time filter   |
| Level  | `Level`        | tea-grid set filter    |
| Msg    | `Msg`          | tea-grid text filter   |
| Attrs  | `Attrs`        | tea-grid text filter   |

The `Attrs` column renders a truncated `key=value, key=value...` summary.

tea-grid's built-in per-column filtering is used for all columns.

Status bar shows: `"{n} records, {m} skipped"` (skipped only if > 0).

### Detail View (`ui/detail.go`)

Displays a single `LogRecord`. Two sub-views, toggled with Tab:

1. **Structured**: Standard fields at top (`time`, `level`, `msg`), then each
   attr key-value pair on its own labeled row.
2. **Raw**: Pretty-printed, syntax-highlighted JSON (the original `RawJSON`
   re-formatted).

Future extensions:
- Collapsible JSON tree view (use an existing bubbletea component if available).
- jq-style query filtering on attrs.

Navigation: Esc returns to the grid view. The detail view supports vertical
scrolling for records with many attributes.

## Error Handling

- Missing or unreadable file path: exit with a usage message before starting
  bubbletea.
- Malformed JSON lines: skipped during loading. Count shown in grid status bar.
- Empty file: grid renders with a "No records" placeholder.

## Testing

- **`log/` package**: Unit tests for JSON parsing — valid records, missing
  standard fields, nested attrs, completely malformed lines.
- **`ui/detail.go`**: Unit test that structured and raw views produce expected
  output for a given `LogRecord`.
- Grid setup is mostly tea-grid configuration; no separate unit tests beyond
  verifying column definitions compile.
- No TUI integration/e2e tests in the initial version.

## Future Extensions

- File tailing / live log streaming
- Inline collapsible attribute sections beneath grid rows
- Collapsible JSON tree view in detail page
- jq-style query language for filtering on attrs
- stdin support (pipe mode)
