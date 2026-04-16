# Log Format Plugin Interface

## Summary

Add a plugin system for log format parsers. Users can define custom log formats
via regex config files or WASM modules, discovered from a directory at startup.
The existing `Format` interface is the sole abstraction -- plugins are just
additional `Format` implementations registered at startup.

## Motivation

Hustle ships with five built-in parsers (JSON, glog, CLF, logfmt, CloudWatch).
Users working with proprietary or niche log formats currently have no way to
add support without forking the codebase. A plugin interface allows:

- Ops teams to describe their custom format in a config file (no code)
- Developers to write a parser in any WASM-targeting language (Rust, Go, C, etc.)
- Contributors to add new built-in formats with less boilerplate

## Design

### Architecture

Everything is a `Format`. There are three backends for creating one:

1. **Go code** (existing) -- implement `Format` directly, register in the
   `Formats` slice in `detect.go`. No changes to this path.
2. **Regex config** -- a `.toml` file describes a regex pattern and field
   mappings. Hustle reads it at startup and constructs a `RegexFormat`.
3. **WASM module** -- a `.wasm` file exports parsing functions. Hustle loads
   it via wazero and wraps it in a `WASMFormat`.

### Format Registration

A new `RegisterFormat` function appends to the existing `Formats` slice:

```go
func RegisterFormat(f Format) {
    Formats = append(Formats, f)
}
```

Built-in formats are the initial slice value. Plugin formats are appended after
them, so built-ins always have detection priority.

### Auto-Detection

`detectFormatFromLines` continues to iterate `Formats` in order and pick the
first format that parses >50% of sample lines. Plugin formats are tried after
all built-ins.

CloudWatch exclusion changes from positional (`Formats[:len(Formats)-1]`) to
interface-based. A `DocumentFormat` marker interface tags formats that need
whole-document parsing rather than line-by-line detection:

```go
type DocumentFormat interface {
    Format
    IsDocumentFormat()
}
```

`detectFormatFromLines` skips any format implementing `DocumentFormat`.

**Future extension:** Confidence-based detection. Formats could optionally
implement a `Detector` interface:

```go
type Detector interface {
    Confidence(sample []string) float64
}
```

Detection would pick the highest-confidence format across all candidates.
Not implemented initially -- recorded here for future reference.

### Discovery and Loading

At startup, hustle scans `~/.config/hustle/formats/`. If the directory does
not exist, the scan is skipped silently. File extension determines the handler:

- `.toml` -- regex format config
- `.wasm` -- WASM plugin module
- Anything else -- ignored (allows README files, etc.)

Loading order within plugins is alphabetical by filename. Users can control
detection priority among plugins via filename prefixes (e.g.
`01-myformat.toml`).

Invoked from `main.go` before loading the log file:

```go
logpkg.LoadPlugins() // scans ~/.config/hustle/formats/
```

**Error handling:** If a plugin fails to load (bad regex, invalid WASM, missing
exports), hustle prints a warning to stderr and continues. A broken plugin
never prevents hustle from starting.

**Name collisions:** If a plugin has the same name as a built-in format, it is
silently ignored. Built-ins always win. `--format name` resolves to the
built-in if there is a collision.

### Regex Format Config

A `.toml` file defines a regex-based format:

```toml
name = "myapp"
pattern = '^(?P<time>\d{4}-\d{2}-\d{2}T[\d:.]+Z?)\s+\[(?P<level>\w+)\]\s+(?P<msg>.*)'
time_format = "2006-01-02T15:04:05.000Z"

[level_map]
"DBG" = "DEBUG"
"INF" = "INFO"
"WRN" = "WARN"
"ERR" = "ERROR"
```

Fields:

- **`name`** (required): format name, used with `--format`.
- **`pattern`** (required): Go regex with named capture groups.
  - `time`, `level`, `msg` map to `LogRecord` fields.
  - All other named groups become `Attrs` entries.
- **`time_format`** (optional): Go time layout string. If omitted, hustle
  tries RFC3339 and common variants.
- **`level_map`** (optional): mapping from captured level strings to canonical
  uppercase values. If omitted, captured values pass through `normalizeLevel`.

Attr values that look like numbers or bools get `inferValue` treatment (same
as logfmt). The regex is compiled once at load time. `RawJSON` is set to the
full input line.

### WASM Plugin ABI

A `.wasm` module exports these functions:

**Required exports:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `name` | `() -> (ptr, len)` | Returns pointer and length of the format name string in module memory |
| `parse` | `(line_ptr, line_len) -> (result_ptr, result_len)` | Parses a line, returns pointer and length of JSON result |
| `alloc` | `(size) -> ptr` | Allocates memory in the module for input data |
| `free` | `(ptr)` | Frees memory previously allocated by `alloc` |

**Parse result format** (JSON in WASM linear memory):

Success:

```json
{
  "ok": true,
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "hello world",
  "attrs": {"key": "value", "count": 42}
}
```

Failure:

```json
{"ok": false, "error": "unrecognized format"}
```

**Memory protocol:** Hustle writes the input line into module memory via
`alloc` + memory write. The module writes its JSON result via `alloc`
internally and returns the pointer from `parse`. Hustle reads the result
and calls `free`. This is the standard wazero convention for passing data
across the WASM boundary.

**Host functions:** None initially. The module receives a line and returns
structured data. Host callbacks (e.g. for nested format delegation) can be
added later without breaking the ABI.

**Runtime:** [wazero](https://github.com/tetratelabs/wazero) -- pure Go
WASM runtime, no CGo dependency. Compiles WASM to native code, so per-call
overhead is microseconds. The JSON serialization of the result is the main
cost but negligible for small `LogRecord` payloads.

**SDK/templates:** Provide starter repos for Rust and Go showing how to build
a `.wasm` format plugin, handling the alloc/free/JSON ceremony so plugin
authors just implement a `parse(line) -> Result<Record, Error>` function.

### Changes to Existing Code

**Modified files:**

- **`log/format.go`** -- add `RegisterFormat`. Add `DocumentFormat` marker
  interface.
- **`log/detect.go`** -- `detectFormatFromLines` skips `DocumentFormat`
  implementations instead of using a positional slice index.
- **`log/loader.go`** -- `Load` checks `DocumentFormat` via type assertion
  instead of `*CloudWatchFormat` directly.
- **`main.go`** -- add `logpkg.LoadPlugins()` call before loading.

**New files:**

- **`log/register.go`** -- `RegisterFormat`, `LoadPlugins` (directory scanning,
  dispatch by extension, error reporting).
- **`log/regex_format.go`** -- `RegexFormat` struct implementing `Format`.
  Compiles regex at construction, extracts named groups per line.
- **`log/wasm_format.go`** -- `WASMFormat` struct implementing `Format`.
  Manages wazero runtime, module instantiation, memory protocol.

**New dependency:**

- `github.com/tetratelabs/wazero` -- pure Go WASM runtime.

**Unchanged:**

- `LogRecord` struct
- `Format` interface (only extended, not modified)
- All five existing parsers
- Grid, jq engine, UI, completion, detail view
- All existing tests
