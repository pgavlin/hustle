package log

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// Load reads a log file, auto-detecting the format or using the given one.
// If format is nil, auto-detection is used.
func Load(path string, format Format) ([]LogRecord, int, Format, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return LoadReader(f, format)
}

// LoadReader reads log lines from a reader, auto-detecting the format or
// using the given one. Returns the records, skip count, detected format,
// and any error.
func LoadReader(r io.Reader, format Format) ([]LogRecord, int, Format, error) {
	// CloudWatch needs document-level parsing, not line-by-line
	if cw, ok := format.(*CloudWatchFormat); ok {
		_ = cw
		return LoadCloudWatchJSON(r)
	}

	var prefixLines []string

	if format == nil {
		var err error
		format, prefixLines, err = DetectFormat(r)
		if err != nil {
			return nil, 0, nil, err
		}
	}

	var records []LogRecord
	skipped := 0

	// Parse pre-sampled lines first
	for _, line := range prefixLines {
		rec, err := format.ParseRecord(line)
		if err != nil {
			skipped++
			continue
		}
		records = append(records, rec)
	}

	// Parse remaining lines from reader
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		rec, err := format.ParseRecord(line)
		if err != nil {
			skipped++
			continue
		}
		records = append(records, rec)
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, nil, fmt.Errorf("read: %w", err)
	}

	return records, skipped, format, nil
}
