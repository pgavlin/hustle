package log

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// Load reads a log file line-by-line, parsing each line as a JSON LogRecord.
// Returns the parsed records, count of skipped (malformed) lines, and any
// file-level error.
func Load(path string) ([]LogRecord, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return LoadReader(f)
}

// LoadReader reads log lines from an io.Reader.
func LoadReader(r io.Reader) ([]LogRecord, int, error) {
	var records []LogRecord
	skipped := 0
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		rec, err := ParseRecord(line)
		if err != nil {
			skipped++
			continue
		}
		records = append(records, rec)
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("read: %w", err)
	}

	return records, skipped, nil
}
