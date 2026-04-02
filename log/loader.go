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

	// Estimate record count from file size to pre-allocate the slice.
	var sizeHint int
	if info, err := f.Stat(); err == nil && info.Size() > 0 {
		// Assume ~200 bytes per line as a rough average.
		sizeHint = int(info.Size()) / 200
	}

	return loadReader(f, format, sizeHint)
}

// LoadReader reads log lines from a reader, auto-detecting the format or
// using the given one. Returns the records, skip count, detected format,
// and any error.
func LoadReader(r io.Reader, format Format) ([]LogRecord, int, Format, error) {
	return loadReader(r, format, 0)
}

func loadReader(r io.Reader, format Format, sizeHint int) ([]LogRecord, int, Format, error) {
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

	records := make([]LogRecord, 0, sizeHint)
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
