package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// newWASMFormat loads a WASM module and returns a Format implementation.
// It auto-detects whether the module uses the export-based ABI (name/parse/alloc/dealloc)
// or the WASI stdio ABI (reads stdin, writes stdout). The fallbackName is used for
// stdio modules that can't self-report their name.
func newWASMFormat(wasmBytes []byte, fallbackName string) (Format, error) {
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("compile WASM module: %w", err)
	}

	// Probe for export-based ABI by checking exported function names.
	exports := compiled.ExportedFunctions()
	_, hasName := exports["name"]
	_, hasParse := exports["parse"]
	_, hasAlloc := exports["alloc"]
	_, hasDealloc := exports["dealloc"]

	if hasName && hasParse && hasAlloc && hasDealloc {
		return newExportWASMFormat(ctx, rt, compiled)
	}

	// Fall back to stdio ABI (e.g. Javy-compiled JS modules).
	return newStdioWASMFormat(rt, compiled, fallbackName)
}

// --- Export-based ABI (Go/Rust plugins) ---

// wasmExportFormat implements Format using exported name/parse/alloc/dealloc functions.
type wasmExportFormat struct {
	name    string
	runtime wazero.Runtime
	module  api.Module
	alloc   api.Function
	dealloc api.Function
	parse   api.Function
}

func unpack(v uint64) (uint32, uint32) {
	return uint32(v), uint32(v >> 32)
}

func newExportWASMFormat(ctx context.Context, rt wazero.Runtime, compiled wazero.CompiledModule) (*wasmExportFormat, error) {
	cfg := wazero.NewModuleConfig().WithStartFunctions()
	mod, err := rt.InstantiateModule(ctx, compiled, cfg)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("instantiate WASM module: %w", err)
	}

	nameFn := mod.ExportedFunction("name")
	parseFn := mod.ExportedFunction("parse")
	allocFn := mod.ExportedFunction("alloc")
	deallocFn := mod.ExportedFunction("dealloc")

	results, err := nameFn.Call(ctx)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("calling name(): %w", err)
	}
	if len(results) != 1 {
		rt.Close(ctx)
		return nil, fmt.Errorf("name() returned %d values, expected 1 (packed ptr|len)", len(results))
	}
	namePtr, nameLen := unpack(results[0])
	nameBytes, ok := mod.Memory().Read(namePtr, nameLen)
	if !ok {
		rt.Close(ctx)
		return nil, fmt.Errorf("failed to read name from WASM memory")
	}

	return &wasmExportFormat{
		name:    string(nameBytes),
		runtime: rt,
		module:  mod,
		alloc:   allocFn,
		dealloc: deallocFn,
		parse:   parseFn,
	}, nil
}

func (f *wasmExportFormat) Name() string { return f.name }

func (f *wasmExportFormat) ParseRecord(line string) (LogRecord, error) {
	ctx := context.Background()

	lineBytes := []byte(line)
	lineSize := uint64(len(lineBytes))
	results, err := f.alloc.Call(ctx, lineSize)
	if err != nil {
		return LogRecord{}, fmt.Errorf("alloc: %w", err)
	}
	linePtr := uint32(results[0])
	defer f.dealloc.Call(ctx, uint64(linePtr), lineSize)

	if !f.module.Memory().Write(linePtr, lineBytes) {
		return LogRecord{}, fmt.Errorf("failed to write line to WASM memory")
	}

	results, err = f.parse.Call(ctx, uint64(linePtr), lineSize)
	if err != nil {
		return LogRecord{}, fmt.Errorf("parse: %w", err)
	}
	if len(results) != 1 {
		return LogRecord{}, fmt.Errorf("parse returned %d values, expected 1", len(results))
	}
	resultPtr, resultLen := unpack(results[0])
	defer f.dealloc.Call(ctx, uint64(resultPtr), uint64(resultLen))

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

func (f *wasmExportFormat) Close() error {
	return f.runtime.Close(context.Background())
}

// --- Stdio-based ABI (Javy/JS plugins) ---

// wasmStdioFormat implements Format by instantiating a WASI module per call,
// passing the line on stdin and reading the JSON result from stdout.
type wasmStdioFormat struct {
	name     string
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	mu       sync.Mutex // wazero runtimes are not safe for concurrent instantiation
}

func newStdioWASMFormat(rt wazero.Runtime, compiled wazero.CompiledModule, name string) (*wasmStdioFormat, error) {
	return &wasmStdioFormat{
		name:     name,
		runtime:  rt,
		compiled: compiled,
	}, nil
}

func (f *wasmStdioFormat) Name() string { return f.name }

func (f *wasmStdioFormat) ParseRecord(line string) (LogRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ctx := context.Background()

	stdin := bytes.NewReader([]byte(line))
	var stdout bytes.Buffer

	var stderr bytes.Buffer
	cfg := wazero.NewModuleConfig().
		WithName("").
		WithStdin(stdin).
		WithStdout(&stdout).
		WithStderr(&stderr)

	mod, err := f.runtime.InstantiateModule(ctx, f.compiled, cfg)
	if err != nil {
		// Modules that call proc_exit(0) return an exit error — that's normal.
		if !strings.Contains(err.Error(), "exit_code(0)") {
			errMsg := stderr.String()
			if errMsg != "" {
				return LogRecord{}, fmt.Errorf("run WASM module: %w\nstderr: %s", err, errMsg)
			}
			return LogRecord{}, fmt.Errorf("run WASM module: %w", err)
		}
	}
	if mod != nil {
		mod.Close(ctx)
	}

	rec, err := parseWASMResult(stdout.Bytes())
	if err != nil {
		return LogRecord{}, err
	}
	rec.RawJSON = line
	return rec, nil
}

func (f *wasmStdioFormat) Close() error {
	return f.runtime.Close(context.Background())
}

// --- Shared result parsing ---

type wasmResult struct {
	OK    bool           `json:"ok"`
	Error string         `json:"error,omitempty"`
	Time  string         `json:"time,omitempty"`
	Level string         `json:"level,omitempty"`
	Msg   string         `json:"msg,omitempty"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

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
