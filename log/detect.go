package log

import (
	"fmt"
	"io"
	"strings"
)

// Formats is the registry of known formats, in auto-detection priority order.
// More specific formats (JSON, glog, CLF) are tried before the permissive
// logfmt parser, which accepts almost anything with key=value pairs.
// CloudWatch is listed for --format lookup but not in line-by-line detection
// (it needs special document-level handling).
var Formats = []Format{
	&JSONFormat{},
	&GlogFormat{},
	&CLFFormat{},
	&LogfmtFormat{},
	&CloudWatchFormat{},
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
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("reading sample: %w", err)
	}
	lines := splitLines(data)
	format, err := detectFormatFromLines(lines)
	return format, lines, err
}

// detectFormatFromLines detects the format from a set of pre-split lines.
func detectFormatFromLines(lines []string) (Format, error) {
	sample := lines
	if len(sample) > detectSampleSize {
		sample = sample[:detectSampleSize]
	}
	if len(sample) == 0 {
		// Empty input — default to JSON
		return &JSONFormat{}, nil
	}

	for _, f := range Formats {
		if _, ok := f.(DocumentFormat); ok {
			continue
		}
		parsed := 0
		for _, line := range sample {
			if _, err := f.ParseRecord(line); err == nil {
				parsed++
			}
		}
		if parsed > len(sample)/2 {
			return f, nil
		}
	}

	return nil, fmt.Errorf(
		"could not detect log format from sample; try --format (%s)",
		strings.Join(FormatNames(), ", "),
	)
}
