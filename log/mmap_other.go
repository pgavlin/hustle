//go:build !unix

package log

import (
	"io"
	"os"
)

// mmapFile falls back to reading the file into a heap-allocated slice on
// platforms that do not support mmap.
func mmapFile(f *os.File, size int64) ([]byte, io.Closer, error) {
	data := make([]byte, size)
	if _, err := io.ReadFull(f, data); err != nil {
		return nil, nil, err
	}
	return data, closerFunc(func() error { return nil }), nil
}
