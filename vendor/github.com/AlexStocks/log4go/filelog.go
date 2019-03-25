// Copyright (C) 2010, Kyle Lemons <kyle@kylelemons.net>.  All rights reserved.

package log4go

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/goext/os"
	"github.com/AlexStocks/goext/strings"
)

// This log writer sends output to a file
type FileLogWriter struct {
	rec chan *LogRecord
	rot chan bool

	// The opened file
	filename string
	file     io.WriteCloser
	//file     *os.File

	// The logging format
	json   bool
	format string

	// File header/trailer
	header, trailer string

	// Rotate at linecount
	maxlines          int
	maxlines_curlines int

	// Rotate at size
	maxsize         int64
	maxsize_cursize int64

	// Rotate daily
	daily          bool
	daily_opendate int

	// Keep old logfiles (.001, .002, etc)
	rotate    bool
	maxbackup int

	sync.Once
}

// This is the FileLogWriter's output method
func (w *FileLogWriter) LogWrite(rec *LogRecord) {
	defer func() {
		if e := recover(); e != nil {
			//js, err := json.Marshal(rec)
			//if err != nil {
			//	fmt.Printf("json.Marshal(rec:%#v) = error{%#v}\n", rec, err)
			//	return
			//}
			fmt.Printf("file log channel has been closed. rec:" + gxstrings.String(rec.JSON()) + "\n")
		}
	}()

	w.rec <- rec
}

func (w *FileLogWriter) Close() {
	w.Once.Do(func() {
		// Wait write coroutine
		for len(w.rec) > 0 {
			time.Sleep(100 * time.Millisecond)
		}

		close(w.rec)

		switch f := w.file.(type) {
		case *os.File:
			f.Sync()
		case *BufFileWriter:
			f.Flush()
		default:
			w.file.Close()
		}
	})
}

// NewFileLogWriter creates a new LogWriter which writes to the given file and
// has rotation enabled if rotate is true and set a memory alignment buffer if
// bufSize is non-zero.
//
// If rotate is true, any time a new log file is opened, the old one is renamed
// with a .### extension to preserve it.  The various Set* methods can be used
// to configure log rotation based on lines, size, and daily.
//
// The standard log-line format is:
//   [%D %T] [%L] (%S) %M
func NewFileLogWriter(fname string, rotate bool, bufSize int) *FileLogWriter {
	w := &FileLogWriter{
		rec:       make(chan *LogRecord, LogBufferLength),
		rot:       make(chan bool),
		filename:  fname,
		json:      false,
		format:    "[%D %T] [%L] (%S) %M",
		rotate:    rotate,
		maxbackup: 999,
	}

	// open the file for the first time
	if err := w.intOpen(bufSize); err != nil {
		fmt.Fprintf(os.Stderr, "FileLogWriter(filename:%q, bufSize:%d): %s\n", w.filename, bufSize, err)
		return nil
	}

	go func() {
		// 关闭文件
		defer func() {
			if w.file != nil {
				fmt.Fprint(w.file, FormatLogRecord(w.trailer, &LogRecord{Created: time.Now()}))
				w.file.Close()
			}
		}()

		for {
			select {
			case <-w.rot: // 外部调用Rotate函数，切割log文件
				if err := w.intRotate(); err != nil {
					fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
					return
				}

			case rec, ok := <-w.rec:
				if !ok { // rec channel关闭了，退出这个log输出goroutine
					return
				}
				// 满足相关设定，切割文件了
				now := time.Now()
				if (w.maxlines > 0 && w.maxlines_curlines >= w.maxlines) ||
					(w.maxsize > 0 && w.maxsize_cursize >= w.maxsize) ||
					(w.daily && now.Day() != w.daily_opendate) {
					if err := w.intRotate(); err != nil {
						fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
						return
					}
				}

				// Perform the write
				// 写log
				var recStr string
				if !w.json {
					recStr = FormatLogRecord(w.format, rec)
				} else {
					//recJson, _ := json.Marshal(rec)
					//recStr = gxstrings.String(recJson)
					recBytes := append(rec.JSON(), gxstrings.Slice(newLine)...)
					recStr = gxstrings.String(recBytes)
				}
				n, err := fmt.Fprint(w.file, recStr)
				if err != nil {
					fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
					return
				}

				// Update the counts
				w.maxlines_curlines++
				w.maxsize_cursize += int64(n)
			}
		}
	}()

	return w
}

// Request that the logs rotate
func (w *FileLogWriter) Rotate() {
	w.rot <- true
}

// If this is called in a threaded context, it MUST be synchronized
// 关闭旧的文件，按照设置rename新的名称，再打开一个文件
func (w *FileLogWriter) intRotate() error {
	// Close any log file that may be open
	// 切割日志文件前，先把当前打开的日志文件关闭
	if w.file != nil {
		fmt.Fprint(w.file, FormatLogRecord(w.trailer, &LogRecord{Created: time.Now()}))
		w.file.Close()
	}

	// If we are keeping log files, move it to the next available number
	// 为旧文件找到一个合适的名称
	if w.rotate {
		_, err := os.Lstat(w.filename)
		if err == nil { // file exists
			// Find the next available number
			num := 1
			fname := ""
			if w.daily && time.Now().Day() != w.daily_opendate {
				yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

				for ; err == nil && num <= w.maxbackup; num++ {
					fname = w.filename + fmt.Sprintf(".%s.%03d", yesterday, num)
					_, err = os.Lstat(fname)
				}
				// return error if the last file checked still existed
				if err == nil {
					return fmt.Errorf("Rotate: Cannot find free log number to rename %s\n", w.filename)
				}
			} else {
				num = w.maxbackup - 1
				for ; num >= 1; num-- {
					fname = w.filename + fmt.Sprintf(".%d", num)
					nfname := w.filename + fmt.Sprintf(".%d", num+1)
					_, err = os.Lstat(fname)
					if err == nil {
						os.Rename(fname, nfname)
					}
				}
			}

			w.file.Close()
			// Rename the file to its newfound home
			err = os.Rename(w.filename, fname)
			if err != nil {
				return fmt.Errorf("Rotate: %s\n", err)
			}
		}
	}

	// Open the log file
	// 打开新的文件
	fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0664)
	if err != nil {
		panic(err) // 打不开文件，说明fd耗完或者磁盘耗尽
		return err
	}
	w.file = fd

	// Set the daily open date to the current date
	now := time.Now()
	w.daily_opendate = now.Day()

	w.maxsize_cursize = 0
	// 防止程序重启创建一个新的日志文件
	if fstat, err := fd.Stat(); nil == err && nil != fstat {
		w.maxsize_cursize = fstat.Size()
		now = fstat.ModTime()
	}
	// initialize rotation values
	w.maxlines_curlines = 0

	fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: now}))

	return nil
}

// If this is called in a threaded context, it MUST be synchronized
func (w *FileLogWriter) intOpen(bufSize int) error {
	// Already opened
	if w.file != nil {
		return nil
	}

	// 创建文件所在的路径
	path := filepath.Dir(w.filename)
	// filepath.Dir("hello.log") = "."
	// filepath.Dir("./hello.log") = "."
	// filepath.Dir("../hello.log") = "..
	if path != "" && path != "." && path != ".." {
		if err := gxos.CreateDir(path); err != nil {
			return err
		}
	}

	// Open the log file
	if bufSize == 0 {
		fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
		if err != nil {
			return err
		}
		w.file = fd
	} else {
		writer := newBufFileWriter(w.filename, bufSize)
		if _, err := writer.open(); err != nil {
			return err
		}
		w.file = writer
	}

	now := time.Now()
	fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: now}))

	// Set the daily open date to the current date
	w.daily_opendate = now.Day()

	// initialize rotation values
	w.maxlines_curlines = 0
	w.maxsize_cursize = 0

	fstat, err := os.Lstat(w.filename)
	if err == nil {
		w.maxsize_cursize = fstat.Size()
	}

	return nil
}

// Set the logging json format (chainable).  Must be called before the first log
// message is written.
func (w *FileLogWriter) SetJson(jsonFormat bool) *FileLogWriter {
	w.json = jsonFormat
	return w
}

// Set the logging format (chainable).  Must be called before the first log
// message is written.
func (w *FileLogWriter) SetFormat(format string) *FileLogWriter {
	w.format = format
	return w
}

// Set the logfile header and footer (chainable).  Must be called before the first log
// message is written.  These are formatted similar to the FormatLogRecord (e.g.
// you can use %D and %T in your header/footer for date and time).
func (w *FileLogWriter) SetHeadFoot(head, foot string) *FileLogWriter {
	w.header, w.trailer = head, foot
	if w.maxlines_curlines == 0 {
		fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: time.Now()}))
	}
	return w
}

// Set rotate at linecount (chainable). Must be called before the first log
// message is written.
func (w *FileLogWriter) SetRotateLines(maxlines int) *FileLogWriter {
	if maxlines > 0 {
		w.maxlines = maxlines
	}
	return w
}

// Set rotate at size (chainable). Must be called before the first log message
// is written.
func (w *FileLogWriter) SetRotateSize(maxsize int64) *FileLogWriter {
	if maxsize > 0 {
		w.maxsize = maxsize
	}
	return w
}

// Set rotate daily (chainable). Must be called before the first log message is
// written.
func (w *FileLogWriter) SetRotateDaily(daily bool) *FileLogWriter {
	w.daily = daily
	return w
}

// Set max backup files. Must be called before the first log message
// is written.
func (w *FileLogWriter) SetRotateMaxBackup(maxbackup int) *FileLogWriter {
	if maxbackup > 0 {
		w.maxbackup = maxbackup
	}
	return w
}

// SetRotate changes whether or not the old logs are kept. (chainable) Must be
// called before the first log message is written.  If rotate is false, the
// files are overwritten; otherwise, they are rotated to another file before the
// new log is opened.
func (w *FileLogWriter) SetRotate(rotate bool) *FileLogWriter {
	w.rotate = rotate
	return w
}

// NewXMLLogWriter is a utility method for creating a FileLogWriter set up to
// output XML record log messages instead of line-based ones.
func NewXMLLogWriter(fname string, rotate bool, bufSize int) *FileLogWriter {
	return NewFileLogWriter(fname, rotate, bufSize).SetFormat(
		`	<record level="%L">
		<timestamp>%D %T</timestamp>
		<source>%S</source>
		<message>%M</message>
	</record>`).SetHeadFoot("<log created=\"%D %T\">", "</log>")
}
