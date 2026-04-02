package log

// Format parses log lines into LogRecords.
type Format interface {
	Name() string
	ParseRecord(line string) (LogRecord, error)
}
