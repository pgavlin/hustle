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

	records, skipped, detected, parseErr := parseData(data, format, 0)
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

	records, skipped, detected, err := parseData(data, format, 0)
	if err != nil {
		return nil, err
	}
	return &LogFile{Records: records, Skipped: skipped, Format: detected}, nil
}

// nextLine extracts the next non-empty line from data, returning the line
// as a zero-copy string and the remaining data. Returns ("", nil) when done.
func nextLine(data []byte) (string, []byte) {
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
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if len(line) > 0 {
			return unsafe.String(unsafe.SliceData(line), len(line)), data
		}
	}
	return "", nil
}

// parseData parses log records from raw data in a single pass.
// If format is nil, it auto-detects from the first detectSampleSize lines.
func parseData(data []byte, format Format, sizeHint int) ([]LogRecord, int, Format, error) {
	if format == nil {
		// Sample first N lines for detection without building a full []string.
		sample := make([]string, 0, detectSampleSize)
		rest := data
		for len(sample) < detectSampleSize {
			line, remaining := nextLine(rest)
			if line == "" {
				break
			}
			sample = append(sample, line)
			rest = remaining
		}

		var err error
		format, err = detectFormatFromLines(sample)
		if err != nil {
			return nil, 0, nil, err
		}
	}

	if sizeHint == 0 {
		sizeHint = bytes.Count(data, []byte{'\n'}) + 1
	}
	records := make([]LogRecord, 0, sizeHint)
	skipped := 0

	// Single pass: scan for \n and parse each line.
	for len(data) > 0 {
		var line string
		line, data = nextLine(data)
		if line == "" {
			break
		}
		rec, err := format.ParseRecord(line)
		if err != nil {
			skipped++
			continue
		}
		records = append(records, rec)
	}

	return records, skipped, format, nil
}
