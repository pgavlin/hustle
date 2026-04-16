package log

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WASMFormat implements Format by delegating to a WASM module that exports
// the hustle plugin ABI: name, parse, alloc, dealloc.
type WASMFormat struct {
	name    string
	runtime wazero.Runtime
	module  api.Module
	alloc   api.Function
	dealloc api.Function
	parse   api.Function
}

// unpack splits a packed u64 return value into (ptr, len).
// Low 32 bits = ptr, high 32 bits = len.
func unpack(v uint64) (uint32, uint32) {
	return uint32(v), uint32(v >> 32)
}

func newWASMFormat(wasmBytes []byte) (*WASMFormat, error) {
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)

	// Instantiate WASI if the module needs it (e.g. TinyGo-compiled modules).
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("compile WASM module: %w", err)
	}

	// Don't run _start automatically — TinyGo's main() calls proc_exit which
	// would close the module. We just need the exported functions.
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

	if nameFn == nil || parseFn == nil || allocFn == nil || deallocFn == nil {
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
		if deallocFn == nil {
			missing += "dealloc "
		}
		rt.Close(ctx)
		return nil, fmt.Errorf("WASM module missing required exports: %s", missing)
	}

	// Call name() to get the format name.
	// Returns a packed u64: low 32 bits = ptr, high 32 bits = len.
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

	return &WASMFormat{
		name:    string(nameBytes),
		runtime: rt,
		module:  mod,
		alloc:   allocFn,
		dealloc: deallocFn,
		parse:   parseFn,
	}, nil
}

func (f *WASMFormat) Name() string { return f.name }

func (f *WASMFormat) ParseRecord(line string) (LogRecord, error) {
	ctx := context.Background()

	// Allocate memory in the module for the input line.
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

	// Call parse(ptr, len) -> packed u64 (result_ptr | result_len << 32).
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
		Attrs: make(Attrs, 0, len(result.Attrs)),
	}
	for k, v := range result.Attrs {
		rec.Attrs.Set(k, v)
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
