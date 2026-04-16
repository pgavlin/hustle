// Example hustle format plugin in Go, compiled with TinyGo.
//
// Parses lines of the form:
//   LEVEL: message key=value key=value ...
//
// Build:
//   tinygo build -o plugin.wasm -target wasm -no-debug main.go
package main

import (
	"unsafe"
)

// Format name returned by the name() export.
const formatName = "example-go"

// pack returns two u32 values packed into a single u64.
// Low 32 bits = ptr, high 32 bits = len.
func pack(ptr, length uint32) uint64 {
	return uint64(ptr) | (uint64(length) << 32)
}

//export name
func name() uint64 {
	ptr := unsafe.Pointer(unsafe.StringData(formatName))
	return pack(uint32(uintptr(ptr)), uint32(len(formatName)))
}

//export parse
func parse(linePtr, lineLen uint32) uint64 {
	line := ptrToString(linePtr, lineLen)
	result := parseLine(line)
	ptr, length := copyToWASM(result)
	return pack(ptr, length)
}

//export alloc
func alloc(size uint32) uint32 {
	buf := make([]byte, size)
	return uint32(uintptr(unsafe.Pointer(unsafe.SliceData(buf))))
}

//export dealloc
func dealloc(ptr uint32, size uint32) {
	// TinyGo's GC handles deallocation; this is a no-op.
}

func ptrToString(ptr, length uint32) string {
	return unsafe.String((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}

func copyToWASM(s string) (uint32, uint32) {
	buf := alloc(uint32(len(s)))
	copy(unsafe.Slice((*byte)(unsafe.Pointer(uintptr(buf))), len(s)), s)
	return buf, uint32(len(s))
}

func parseLine(line string) string {
	// Find ":" separator between level and message
	colonIdx := -1
	for i := 0; i < len(line); i++ {
		if line[i] == ':' {
			colonIdx = i
			break
		}
	}
	if colonIdx < 0 || colonIdx+2 > len(line) {
		return `{"ok":false,"error":"expected LEVEL: message"}`
	}

	level := line[:colonIdx]
	if !isLevel(level) {
		return `{"ok":false,"error":"unknown level"}`
	}

	rest := line[colonIdx+2:] // skip ": "

	msg, attrs := splitMsgAttrs(rest)

	// Build JSON manually to avoid importing encoding/json (keeps WASM small)
	var buf []byte
	buf = append(buf, `{"ok":true,"level":"`...)
	buf = append(buf, level...)
	buf = append(buf, `","msg":"`...)
	buf = appendJSONEscaped(buf, msg)
	buf = append(buf, '"')

	if len(attrs) > 0 {
		buf = append(buf, `,"attrs":{`...)
		first := true
		for _, kv := range attrs {
			if !first {
				buf = append(buf, ',')
			}
			first = false
			buf = append(buf, '"')
			buf = append(buf, kv.key...)
			buf = append(buf, `":"`...)
			buf = appendJSONEscaped(buf, kv.val)
			buf = append(buf, '"')
		}
		buf = append(buf, '}')
	}

	buf = append(buf, '}')
	return string(buf)
}

type keyVal struct {
	key, val string
}

func isLevel(s string) bool {
	switch s {
	case "TRACE", "DEBUG", "INFO", "WARN", "WARNING", "ERROR", "FATAL", "PANIC":
		return true
	}
	return false
}

// splitMsgAttrs splits "message text key=val key2=val2" into message and attrs.
// Attrs are contiguous key=value tokens at the end of the string.
func splitMsgAttrs(s string) (string, []keyVal) {
	tokens := splitTokens(s)
	attrStart := len(tokens)
	for i := len(tokens) - 1; i >= 0; i-- {
		if !containsByte(tokens[i], '=') {
			break
		}
		attrStart = i
	}

	var attrs []keyVal
	for i := attrStart; i < len(tokens); i++ {
		eq := indexByte(tokens[i], '=')
		if eq > 0 {
			attrs = append(attrs, keyVal{
				key: tokens[i][:eq],
				val: tokens[i][eq+1:],
			})
		}
	}

	var msg string
	if attrStart > 0 {
		end := 0
		count := 0
		for i := 0; i < len(s); i++ {
			if s[i] == ' ' {
				count++
				if count == attrStart {
					end = i
					break
				}
			}
		}
		if end == 0 && attrStart >= len(tokens) {
			msg = s
		} else {
			msg = s[:end]
		}
	}

	return msg, attrs
}

func splitTokens(s string) []string {
	var tokens []string
	start := 0
	inToken := false
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			if inToken {
				tokens = append(tokens, s[start:i])
				inToken = false
			}
		} else {
			if !inToken {
				start = i
				inToken = true
			}
		}
	}
	if inToken {
		tokens = append(tokens, s[start:])
	}
	return tokens
}

func containsByte(s string, b byte) bool {
	return indexByte(s, b) >= 0
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func appendJSONEscaped(buf []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			buf = append(buf, '\\', '"')
		case '\\':
			buf = append(buf, '\\', '\\')
		case '\n':
			buf = append(buf, '\\', 'n')
		case '\r':
			buf = append(buf, '\\', 'r')
		case '\t':
			buf = append(buf, '\\', 't')
		default:
			buf = append(buf, s[i])
		}
	}
	return buf
}

func main() {}
