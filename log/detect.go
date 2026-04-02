package log

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Formats is the registry of known formats, in auto-detection priority order.
var Formats = []Format{
	&JSONFormat{},
	&LogfmtFormat{},
	&CLFFormat{},
}

// FormatByName returns the format with the given name, or nil.
func FormatByName(name string) Format {
	for _, f := range Formats {
		if f.Name() == name {
			return f
		}
	}
	return nil
}

// FormatNames returns the names of all known formats.
func FormatNames() []string {
	names := make([]string, len(Formats))
	for i, f := range Formats {
		names[i] = f.Name()
	}
	return names
}

const detectSampleSize = 10

// DetectFormat reads up to detectSampleSize non-empty lines from the reader
// and tries each format. Returns the first format that parses >50% of the
// sample lines, along with the sampled lines (so the caller doesn't lose them).
func DetectFormat(r io.Reader) (Format, []string, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() && len(lines) < detectSampleSize {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("reading sample: %w", err)
	}
	if len(lines) == 0 {
		// Empty input — default to JSON
		return &JSONFormat{}, nil, nil
	}

	for _, f := range Formats {
		parsed := 0
		for _, line := range lines {
			if _, err := f.ParseRecord(line); err == nil {
				parsed++
			}
		}
		if parsed > len(lines)/2 {
			return f, lines, nil
		}
	}

	return nil, lines, fmt.Errorf(
		"could not detect log format from sample; try --format (%s)",
		strings.Join(FormatNames(), ", "),
	)
}
