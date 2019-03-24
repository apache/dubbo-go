/******************************************************
# DESC    :
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2018-03-25 19:35
# FILE    : buffile.go
******************************************************/

package log4go

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
)

import (
	"github.com/AlexStocks/goext/strings"
)

type BufFileWriter struct {
	filename string
	bufSize  int

	sync.RWMutex

	file      *os.File
	bufWriter *bufio.Writer
	writer    io.Writer
}

func newBufFileWriter(filename string, bufSize int) *BufFileWriter {
	return &BufFileWriter{
		filename: filename,
		bufSize:  bufSize,
	}
}

func (w *BufFileWriter) Reset(filename string, bufSize int) {
	w.Close()
	w.filename = filename
	w.bufSize = bufSize
}

func (w *BufFileWriter) open() (*os.File, error) {
	w.Lock()
	defer w.Unlock()

	fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return nil, err
	}

	w.file = fd
	w.writer = w.file

	if w.bufSize > 0 {
		w.bufWriter = bufio.NewWriterSize(w.file, w.bufSize)
		w.writer = w.bufWriter
	}
	return fd, nil
}

func (w *BufFileWriter) Close() error {
	w.Flush()

	w.Lock()
	defer w.Unlock()

	if w.file == nil {
		return nil
	}

	w.file.Close()
	w.file = nil
	w.writer = nil
	w.bufWriter = nil

	return nil
}

func (w *BufFileWriter) Flush() {
	w.Lock()
	defer w.Unlock()

	if w.bufWriter != nil {
		w.bufWriter.Flush()
		return
	}
	if w.file != nil {
		w.file.Sync()
	}
}

func (w *BufFileWriter) Seek(offset int64, whence int) (int64, error) {
	w.Lock()
	defer w.Unlock()

	if w.file != nil {
		return w.file.Seek(offset, whence)
	}

	fi, err := os.Lstat(w.filename)
	if err != nil {
		return 0, err
	}

	return fi.Size(), nil
}

func (w *BufFileWriter) Write(p []byte) (int, error) {
	if w.file == nil {
		_, err := w.open()
		if err != nil {
			return 0, err
		}
	}

	w.Lock()
	defer w.Unlock()
	return fmt.Fprint(w.writer, gxstrings.String(p))
}
