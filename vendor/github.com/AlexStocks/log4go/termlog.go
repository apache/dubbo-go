// Copyright (C) 2010, Kyle Lemons <kyle@kylelemons.net>.  All rights reserved.

package log4go

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/goext/strings"
)

var stdout io.Writer = os.Stdout

// This is the standard writer that prints to standard output.
type ConsoleLogWriter struct {
	json   bool
	format string
	lock   sync.Mutex
	w      chan *LogRecord
	sync.Once
}

// This creates a new ConsoleLogWriter
func NewConsoleLogWriter(json bool) *ConsoleLogWriter {
	consoleWriter := &ConsoleLogWriter{
		json:   json,
		format: "[%T %D] [%L] (%S) %M",
		w:      make(chan *LogRecord, LogBufferLength),
	}
	go consoleWriter.run(stdout)
	return consoleWriter
}

func (c *ConsoleLogWriter) SetJson(json bool) {
	c.json = json
}

func (c *ConsoleLogWriter) SetFormat(format string) {
	c.format = format
}

func (c *ConsoleLogWriter) run(out io.Writer) {
	//for rec := range c.w {
	//	fmt.Fprint(out, FormatLogRecord(c.format, rec))
	//}
	var logString string
	for rec := range c.w {
		if !c.json {
			logString = FormatLogRecord(c.format, rec)
		} else {
			recBytes := append(rec.JSON(), gxstrings.Slice(newLine)...)
			logString = gxstrings.String(recBytes)
		}
		switch rec.Level {
		case FINEST, FINE, DEBUG:
			cDebug(out, logString)
		case TRACE:
			cTrace(out, logString)
		case INFO:
			cInfo(out, logString)
		case WARNING:
			cWarn(out, logString)
		case ERROR:
			cError(out, logString)
		case CRITICAL:
			cCritical(out, logString)
		}
	}
}

// This is the ConsoleLogWriter's output method.  This will block if the output
// buffer is full.
func (c *ConsoleLogWriter) LogWrite(rec *LogRecord) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf(FormatLogRecord("term log channel has been closed."+c.format, rec))
		}
	}()

	c.w <- rec
}

// Close stops the logger from sending messages to standard output.  Attempts to
// send log messages to this logger after a Close have undefined behavior.
func (c *ConsoleLogWriter) Close() {
	c.Once.Do(func() {
		close(c.w)
		c.w = nil
		time.Sleep(50 * time.Millisecond) // Try to give console I/O time to complete
	})
}
