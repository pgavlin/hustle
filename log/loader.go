package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"unsafe"
)

// LogFile holds the records loaded from a log file.
// Call Close when done to release resources (e.g. munmap the backing file).
type LogFile struct {
	Records []LogRecord
	Skipped int
	Format  Format
	closer  io.Closer
}

// Close releases resources held by the LogFile.
// After Close, the strings inside Records must not be accessed.
func (lf *LogFile) Close() error {
	if lf.closer != nil {
		return lf.closer.Close()
	}
	return nil
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

// Load reads a log file, auto-detecting the format or using the given one.
// The file is mmap'd on supported platforms so that the OS pages it in
// lazily without loading the entire file into the Go heap.
// Call Close on the returned LogFile to release the mapping.
func Load(path string, format Format) (*LogFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	// Document formats need special handling; stream them directly.
	if _, ok := format.(DocumentFormat); ok {
		records, skipped, fmt, err := LoadCloudWatchJSON(f)
		if err != nil {
			return nil, err
		}
		return &LogFile{Records: records, Skipped: skipped, Format: fmt}, nil
	}

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	if info.Size() == 0 {
		return &LogFile{Format: &JSONFormat{}}, nil
	}

	data, closer, err := mmapFile(f, info.Size())
	if err != nil {
		return nil, fmt.Errorf("mmap %s: %w", path, err)
	}

	records, skipped, detected, parseErr := loadLines(splitLines(data), format, len(data)/200)
	if parseErr != nil {
		closer.Close()
		return nil, parseErr
	}
	return &LogFile{Records: records, Skipped: skipped, Format: detected, closer: closer}, nil
}

// LoadReader reads log lines from a reader, auto-detecting the format or
// using the given one. All data is read into memory at once.
func LoadReader(r io.Reader, format Format) (*LogFile, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	// Document formats need special handling.
	if _, ok := format.(DocumentFormat); ok {
		records, skipped, fmt, err := LoadCloudWatchJSON(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		return &LogFile{Records: records, Skipped: skipped, Format: fmt}, nil
	}

	records, skipped, detected, err := loadLines(splitLines(data), format, 0)
	if err != nil {
		return nil, err
	}
	return &LogFile{Records: records, Skipped: skipped, Format: detected}, nil
}

// splitLines returns zero-copy string views of each non-empty line in data.
// All returned strings share the backing array of data; the caller must keep
// data alive (e.g. via an active mmap) for as long as the strings are in use.
func splitLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	lines := make([]string, 0, len(data)/200+1)
	for len(data) > 0 {
		i := bytes.IndexByte(data, '\n')
		var line []byte
		if i < 0 {
			line = data
			data = nil
		} else {
			line = data[:i]
			data = data[i+1:]
		}
		// Strip trailing \r (CRLF)
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if len(line) > 0 {
			lines = append(lines, unsafe.String(unsafe.SliceData(line), len(line)))
		}
	}
	return lines
}

// loadLines parses pre-split lines using the given format.
// If format is nil, it auto-detects from the first detectSampleSize lines.
func loadLines(lines []string, format Format, sizeHint int) ([]LogRecord, int, Format, error) {
	if format == nil {
		var err error
		format, err = detectFormatFromLines(lines)
		if err != nil {
			return nil, 0, nil, err
		}
	}

	if sizeHint == 0 {
		sizeHint = len(lines)
	}
	records := make([]LogRecord, 0, sizeHint)
	skipped := 0
	for _, line := range lines {
		rec, err := format.ParseRecord(line)
		if err != nil {
			skipped++
			continue
		}
		records = append(records, rec)
	}
	return records, skipped, format, nil
}
