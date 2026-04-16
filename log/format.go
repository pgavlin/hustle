package log

// Format parses log lines into LogRecords.
type Format interface {
	Name() string
	ParseRecord(line string) (LogRecord, error)
}

// DocumentFormat is implemented by formats that parse whole documents
// rather than individual lines (e.g. CloudWatch JSON). Document formats
// are skipped during line-by-line auto-detection.
type DocumentFormat interface {
	Format
	IsDocumentFormat()
}

// RegisterFormat adds a format to the registry. Plugin formats are
// appended after built-ins, so built-in formats always have detection
// priority.
func RegisterFormat(f Format) {
	Formats = append(Formats, f)
}
