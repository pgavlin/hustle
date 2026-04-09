//go:build unix

package log

import (
	"io"
	"os"
	"syscall"
)

func mmapFile(f *os.File, size int64) ([]byte, io.Closer, error) {
	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, nil, err
	}
	return data, closerFunc(func() error {
		return syscall.Munmap(data)
	}), nil
}
