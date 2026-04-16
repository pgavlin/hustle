# hustle

A fast terminal viewer for structured logs.

![Demo](demo.gif)

## Features

- **Multi-format support** -- JSON (slog, zap, zerolog, logrus, bunyan), glog, CLF, logfmt, CloudWatch
- **Auto-detection** -- automatically identifies the log format from the first few lines
- **Quick filter** -- type `/` to search across all fields
- **jq filter** -- type `:` to apply jq expressions with tab completion
- **Detail view** -- press Enter on any row to inspect all fields
- **Sorting** -- sort by any column with keyboard shortcuts
- **Plugin system** -- add custom parsers via regex config files or WASM modules

## Install

```
go install github.com/pgavlin/hustle@latest
```

## Usage

```
hustle [options] [logfile]
```

Read from a file:
```
hustle app.log
```

Read from stdin:
```
kubectl logs my-pod | hustle
cat /var/log/app.json | hustle
```

### Options

| Flag | Description |
|------|-------------|
| `--format`, `-f` | Log format: `auto` (default), `json`, `glog`, `clf`, `logfmt`, `cloudwatch` |
| `--cpuprofile` | Write CPU profile to this path |
| `--memprofile` | Write memory profile to this path on exit |

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| `j` / `Down` | Move down |
| `k` / `Up` | Move up |
| `g` | Go to top |
| `G` | Go to bottom |
| `Enter` | View record details / sort (on header) |
| `Escape` | Back to grid / dismiss |
| `/` | Quick filter (search all columns) |
| `:` | jq filter expression |
| `Tab` | Cycle completions (in filter) |
| `Shift+Tab` | Cycle completions backwards |
| `s` | Sort by focused column |
| `S` | Multi-column sort |
| `?` | Toggle help overlay |
| `q` / `Ctrl+C` | Quit |

## Formats

hustle auto-detects these formats:

| Format | Example |
|--------|---------|
| **json** | `{"time":"...","level":"INFO","msg":"hello","key":"value"}` |
| **glog** | `I0615 14:00:03.072000 1042 server.go:123] message` |
| **clf** | `127.0.0.1 - - [15/Jun/2024:14:00:03 +0000] "GET / HTTP/1.1" 200 512` |
| **logfmt** | `time=2024-06-15T14:00:03Z level=info msg="hello" key=value` |
| **cloudwatch** | `aws logs tail` output or `aws logs filter-log-events` JSON |

### CloudWatch

CloudWatch logs require `--format cloudwatch` because they use document-level
parsing rather than line-by-line detection. Two input formats are supported:

- **`aws logs tail`** -- line-by-line output (`timestamp stream message`). The
  inner message is auto-detected as JSON, logfmt, glog, or plain text.
- **`aws logs filter-log-events`** -- JSON document with an `events` array.
  Pipe the full JSON output to hustle.

```
aws logs tail /my/log-group --follow | hustle --format cloudwatch
aws logs filter-log-events --log-group-name /my/log-group | hustle --format cloudwatch
```

## Plugins

Custom log formats can be added by placing files in `~/.config/hustle/formats/`:

### Regex plugins (.toml)

```toml
name = "myapp"
pattern = '^(?P<time>\d{4}-\d{2}-\d{2}T[\d:.]+Z)\s+\[(?P<level>\w+)\]\s+(?P<msg>.*)'
time_format = "2006-01-02T15:04:05.000Z"

[level_map]
"DBG" = "DEBUG"
"INF" = "INFO"
"WRN" = "WARN"
"ERR" = "ERROR"
```

Named capture groups `time`, `level`, and `msg` map to log record fields. All other named groups become attributes.

### WASM plugins (.wasm)

WASM modules export four functions:

| Function | Signature | Description |
|----------|-----------|-------------|
| `name` | `() -> u64` | Returns packed (ptr, len) of the format name |
| `parse` | `(ptr, len) -> u64` | Parses a line, returns packed (ptr, len) of JSON result |
| `alloc` | `(size) -> ptr` | Allocates memory for input |
| `dealloc` | `(ptr, size)` | Frees allocated memory |

The parse function returns JSON: `{"ok":true,"level":"INFO","msg":"hello","attrs":{"key":"value"}}`.

See `examples/wasm-plugin-go/`, `examples/wasm-plugin-rust/`, and `examples/wasm-plugin-zig/` for complete examples.

## Performance

Benchmarks on Apple M4 Max, 100K lines (~11-14 MB):

| Format | Load time | Throughput | Memory |
|--------|-----------|------------|--------|
| glog | 31 ms | 371 MB/s | 48 MB |
| json | 41 ms | 338 MB/s | 54 MB |
| logfmt | 51 ms | 230 MB/s | 97 MB |

Line splitting runs at 6 GB/s with 4 allocations regardless of format.

## License

[MIT](LICENSE)
