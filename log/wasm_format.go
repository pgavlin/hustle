package log

import (
	"context"
	"encoding/json"
	"fmt"
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

	lineBytes := []byte(line)
	results, err := f.alloc.Call(ctx, uint64(len(lineBytes)))
	if err != nil {
		return LogRecord{}, fmt.Errorf("alloc: %w", err)
	}
	linePtr := uint32(results[0])
	defer f.free.Call(ctx, uint64(linePtr))

	if !f.module.Memory().Write(linePtr, lineBytes) {
		return LogRecord{}, fmt.Errorf("failed to write line to WASM memory")
	}

	results, err = f.parse.Call(ctx, uint64(linePtr), uint64(len(lineBytes)))
	if err != nil {
		return LogRecord{}, fmt.Errorf("parse: %w", err)
	}
	if len(results) != 2 {
		return LogRecord{}, fmt.Errorf("parse returned %d values, expected 2", len(results))
	}
	resultPtr, resultLen := uint32(results[0]), uint32(results[1])
	defer f.free.Call(ctx, uint64(resultPtr))

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

func (f *WASMFormat) Close() error {
	return f.runtime.Close(context.Background())
}

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
