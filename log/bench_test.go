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

// BenchmarkLoad benchmarks the full load path: scan + detect + parse.
func BenchmarkLoad(b *testing.B) {
	const nLines = 100_000

	b.Run("glog", func(b *testing.B) {
		data := []byte(strings.Join(generateGlogLines(nLines), "\n"))
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			parseData(data, &GlogFormat{}, 0)
		}
	})

	b.Run("json", func(b *testing.B) {
		data := []byte(strings.Join(generateJSONLines(nLines), "\n"))
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			parseData(data, &JSONFormat{}, 0)
		}
	})

	b.Run("logfmt", func(b *testing.B) {
		data := []byte(strings.Join(generateLogfmtLines(nLines), "\n"))
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			parseData(data, &LogfmtFormat{}, 0)
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
