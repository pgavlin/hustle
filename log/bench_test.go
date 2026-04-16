package log

import (
	"strings"
	"testing"
)

// BenchmarkParseRecord benchmarks parsing individual lines for each format.
func BenchmarkParseRecord(b *testing.B) {
	const nLines = 1000

	b.Run("glog", func(b *testing.B) {
		lines := generateGlogLines(nLines)
		f := &GlogFormat{}
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			for _, line := range lines {
				f.ParseRecord(line)
			}
		}
	})

	b.Run("json", func(b *testing.B) {
		lines := generateJSONLines(nLines)
		f := &JSONFormat{}
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			for _, line := range lines {
				f.ParseRecord(line)
			}
		}
	})

	b.Run("logfmt", func(b *testing.B) {
		lines := generateLogfmtLines(nLines)
		f := &LogfmtFormat{}
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			for _, line := range lines {
				f.ParseRecord(line)
			}
		}
	})
}

// BenchmarkLoad benchmarks the full load path: splitLines + detect + parse.
func BenchmarkLoad(b *testing.B) {
	const nLines = 100_000

	b.Run("glog", func(b *testing.B) {
		data := []byte(strings.Join(generateGlogLines(nLines), "\n"))
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			lines := splitLines(data)
			loadLines(lines, &GlogFormat{}, len(lines))
		}
	})

	b.Run("json", func(b *testing.B) {
		data := []byte(strings.Join(generateJSONLines(nLines), "\n"))
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			lines := splitLines(data)
			loadLines(lines, &JSONFormat{}, len(lines))
		}
	})

	b.Run("logfmt", func(b *testing.B) {
		data := []byte(strings.Join(generateLogfmtLines(nLines), "\n"))
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			lines := splitLines(data)
			loadLines(lines, &LogfmtFormat{}, len(lines))
		}
	})
}

// BenchmarkDetect benchmarks format auto-detection.
func BenchmarkDetect(b *testing.B) {
	b.Run("glog", func(b *testing.B) {
		lines := generateGlogLines(100)
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			detectFormatFromLines(lines)
		}
	})

	b.Run("json", func(b *testing.B) {
		lines := generateJSONLines(100)
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			detectFormatFromLines(lines)
		}
	})
}

// BenchmarkSplitLines benchmarks zero-copy line splitting.
func BenchmarkSplitLines(b *testing.B) {
	const nLines = 100_000
	data := []byte(strings.Join(generateGlogLines(nLines), "\n"))
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		splitLines(data)
	}
}
