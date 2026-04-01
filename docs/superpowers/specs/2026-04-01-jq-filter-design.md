# Hustle: jq Filter Integration

## Overview

Add jq-style query filtering to hustle. Users can enter a jq expression via a
modal input; records where the expression produces `false`, `null`, or empty
output are hidden from the grid. The jq filter composes with tea-grid's
existing per-column filters (both must pass for a record to be shown).

## Dependencies

- `github.com/itchyny/gojq` — pure Go jq implementation, operates directly on
  `map[string]any` values.

## Architecture

**Approach**: Use tea-grid's `WithExternalFilter` to apply the compiled jq
predicate. The gojq dependency is isolated in a `filter` package; the UI never
touches gojq directly.

### New Files

| File | Responsibility |
|------|---------------|
| `filter/jq.go` | Compile jq expressions, evaluate against LogRecord |
| `filter/jq_test.go` | Tests for compilation and evaluation |
| `ui/jqinput.go` | Modal text input for entering jq expressions |

### Modified Files

| File | Change |
|------|--------|
| `ui/grid.go` | Accept optional external filter function in `newLogGrid` |
| `ui/app.go` | Keybinding to open modal, store compiled filter, rebuild grid with ExternalFilter |

## Filter Package (`filter/jq.go`)

Public API:

```go
type JQFilter struct { ... } // wraps compiled gojq.Code
func Compile(expr string) (*JQFilter, error)
func (f *JQFilter) Match(rec log.LogRecord) bool
```

`Match` reconstructs the full record as a `map[string]any`:

```go
map[string]any{
    "time":  rec.Time.Format(time.RFC3339Nano),
    "level": rec.Level,
    "msg":   rec.Msg,
    // + all entries from rec.Attrs merged in
}
```

Then runs the compiled jq code against this map. Result interpretation:
- `false`, `null`, or no output → return false (row hidden)
- Any other value (including `true`, strings, numbers) → return true (row shown)

This matches jq's `select()` semantics: `select(.level == "ERROR")` produces
the input on match and empty on mismatch.

## Modal Input (`ui/jqinput.go`)

A `JQInputModel` that wraps a text input field with an error display line.

- Shows the current expression text (or empty for new filter)
- Error line below the input shows compilation errors
- Enter: attempt to compile the expression. If valid, return it. If invalid,
  show the error and stay in the modal.
- Esc: cancel, keep previous filter unchanged
- Entering an empty expression and pressing Enter clears the active filter

## App Model Changes (`ui/app.go`)

New state:
- `jqFilter *filter.JQFilter` — the active compiled filter (nil = no filter)
- `jqExpr string` — the current expression text (for display and re-editing)
- `jqInput JQInputModel` — the modal input model

New view: `viewJQInput` added to the view enum.

Behavior:
- `j` key (when in grid view and grid is not filtering): enter `viewJQInput`,
  initialize the modal with the current `jqExpr`
- When the modal returns a valid expression: compile it, store in `jqFilter`
  and `jqExpr`, rebuild the grid with `WithExternalFilter` using the filter's
  `Match` method
- When the modal returns empty: clear `jqFilter` and `jqExpr`, rebuild grid
  without external filter
- When the modal is cancelled (Esc): return to grid, no changes

Status bar: when a jq filter is active, append `| jq: <expr>` to the status
bar text.

## Grid Changes (`ui/grid.go`)

`newLogGrid` gains an optional external filter parameter:

```go
func newLogGrid(records []log.LogRecord, width, height int, extFilter func(log.LogRecord) bool) grid.Model[log.LogRecord]
```

When `extFilter` is non-nil, pass it to `grid.WithExternalFilter`. When nil,
omit the option.

## Error Handling

- Invalid jq syntax: shown as an error in the modal. The user must fix or
  cancel. The previous filter remains active.
- Runtime jq errors (e.g. type errors during evaluation): treated as
  non-matching (row hidden). This is consistent with jq's behavior where
  errors in `select()` produce empty output.

## Testing

`filter/jq_test.go`:
- Valid expression compiles without error
- Invalid syntax returns compile error
- `select(.level == "ERROR")` matches ERROR records, hides others
- `.level == "INFO"` (boolean result) matches INFO records
- Expression accessing nested attrs works
- Expression on record with empty attrs doesn't panic
- Empty/null jq output hides the record
- Runtime type errors hide the record (non-match)
