# Multi-Format Log Parsing

## Overview

Add support for multiple log formats beyond slog JSON. Hustle auto-detects the
format by sampling the first 10 non-empty lines, or accepts an explicit
`--format` flag. All formats parse into the existing `LogRecord` structure.

## Formats

### 1. JSON (auto-detecting field names)

Covers slog, zap, zerolog, logrus, bunyan, docker JSON, and generic NDJSON.
Tries common field name variants:

| Field | Candidates (tried in order) |
|-------|----------------------------|
| Time | `time`, `ts`, `timestamp`, `@timestamp` |
| Level | `level`, `severity`, `lvl` |
| Message | `msg`, `message`, `log` |

Time parsing:
- String values: RFC3339 / RFC3339Nano
- Numeric values (zap `ts`): unix timestamp (seconds as float64)

Level normalization:
- String levels uppercased: `"info"` → `"INFO"`
- Bunyan numeric levels: 10→TRACE, 20→DEBUG, 30→INFO, 40→WARN, 50→ERROR, 60→FATAL

All remaining fields go into `Attrs`.

### 2. logfmt

Key-value pairs separated by spaces. Values optionally quoted.

Example: `time=2024-01-15T10:30:00Z level=info msg="server started" port=8080`

Parsing:
- Split on unquoted spaces into `key=value` pairs
- `time`/`ts`/`timestamp` → `Time` (RFC3339)
- `level`/`lvl`/`severity` → `Level`
- `msg`/`message` → `Msg`
- Everything else → `Attrs`
- Bare values (no `=`) are ignored
- Numeric-looking values parsed as `float64`, `"true"`/`"false"` as `bool`

### 3. Combined Log Format (CLF)

Standard nginx/Apache access log format.

Example: `127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /path HTTP/1.0" 200 2326 "http://example.com" "Mozilla/5.0"`

Parsed via regex into:
- `Time`: from the `[datetime]` field
- `Level`: derived from status code (2xx→INFO, 3xx→INFO, 4xx→WARN, 5xx→ERROR)
- `Msg`: the request line (e.g. `GET /path HTTP/1.0`)
- `Attrs`: `remote_addr`, `user`, `status`, `bytes`, `referer`, `user_agent`

## Auto-Detection

Sample the first 10 non-empty lines. Try each format in order: JSON → logfmt →
CLF. The first format that successfully parses >50% of the sample lines is
selected. If no format reaches the threshold, exit with an error suggesting
`--format`.

When `--format` is specified, skip detection and use the given format directly.

## CLI

```
--format, -f   Log format: auto, json, logfmt, clf (default: auto)
```

## Architecture

### Format Interface

```go
type Format interface {
    Name() string
    ParseRecord(line string) (LogRecord, error)
}
```

### File Layout

| File | Responsibility |
|------|---------------|
| `log/format.go` | Format interface, format registry |
| `log/json.go` | JSON format with auto-detecting field names |
| `log/logfmt.go` | logfmt parser |
| `log/clf.go` | Combined Log Format parser |
| `log/detect.go` | Auto-detection (sample lines, try formats) |
| `log/loader.go` | Updated: Load/LoadReader accept optional Format |
| `log/record.go` | LogRecord unchanged; ParseRecord wraps JSON format |
| `main.go` | Add `--format` flag, pass to loader |

### Testing

Each format gets its own test file:
- `log/json_test.go` — slog, zap (unix ts), zerolog, logrus, bunyan (numeric levels), docker JSON
- `log/logfmt_test.go` — standard logfmt, quoted values, missing fields, numeric values
- `log/clf_test.go` — combined log format, common log format (without referer/UA)
- `log/detect_test.go` — auto-detection with mixed/ambiguous input

### Error Handling

- Lines that fail to parse are skipped (counted in `skipped`)
- Auto-detection failure: exit with error suggesting `--format`
- Invalid `--format` value: exit with error listing valid formats
