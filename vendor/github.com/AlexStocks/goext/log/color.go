// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// package gxlog is based on log4go.
// color.go provides colorful terminal log output functions.
package gxlog

import (
	"fmt"
	"os"
)

import (
	"github.com/mattn/go-isatty"
)

var (
	// Normal colors
	// NORMAL   = []byte{'\033', '0', 'm'}
	NORMAL   = []byte{'\033', '0'}
	NBlack   = []byte{'\033', '[', '3', '0', 'm'}
	NRed     = []byte{'\033', '[', '3', '1', 'm'}
	NGreen   = []byte{'\033', '[', '3', '2', 'm'}
	NYellow  = []byte{'\033', '[', '3', '3', 'm'}
	NBlue    = []byte{'\033', '[', '3', '4', 'm'}
	NMagenta = []byte{'\033', '[', '3', '5', 'm'}
	NCyan    = []byte{'\033', '[', '3', '6', 'm'}
	NWhite   = []byte{'\033', '[', '3', '7', 'm'}
	// Bright colors
	BBlack                    = []byte{'\033', '[', '3', '0', ';', '1', 'm'}
	BRed                      = []byte{'\033', '[', '3', '1', ';', '1', 'm'}
	BGreen                    = []byte{'\033', '[', '3', '2', ';', '1', 'm'}
	BYellow                   = []byte{'\033', '[', '3', '3', ';', '1', 'm'}
	BBlue                     = []byte{'\033', '[', '3', '4', ';', '1', 'm'}
	BMagenta                  = []byte{'\033', '[', '3', '5', ';', '1', 'm'}
	BCyan                     = []byte{'\033', '[', '3', '6', ';', '1', 'm'}
	BWhite                    = []byte{'\033', '[', '3', '7', ';', '1', 'm'}
	UnderlineTwinkleHighLight = []byte{'\033', '[', '1', ';', '6', ';', '4', '0', 'm'}

	reset = []byte{'\033', '[', '0', 'm'}
)

func CPrintf(color []byte, format string, args ...interface{}) {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Fprintf(os.Stdout, string(color)+funcFileLine()+fmt.Sprintf(format, args...)+string(reset))
	} else {
		fmt.Fprintf(os.Stdout, fmt.Sprintf(format, args...))
	}
}

func CPrintfln(color []byte, format string, args ...interface{}) {
	logStr := fmt.Sprintf(format, args...)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Fprintf(os.Stdout, string(color)+funcFileLine()+"%s"+string(reset)+"\n", logStr)
	} else {
		fmt.Fprintf(os.Stdout, "%s\n", logStr)
	}
}

func CEPrintf(color []byte, format string, args ...interface{}) {
	logStr := fmt.Sprintf(format, args...)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Fprintf(os.Stderr, string(color)+funcFileLine()+"%s"+string(reset), logStr)
	} else {
		fmt.Fprintf(os.Stderr, "%s", logStr)
	}
}

func CEPrintfln(color []byte, format string, args ...interface{}) {
	logStr := fmt.Sprintf(format, args...)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Fprintf(os.Stderr, string(color)+funcFileLine()+"%s"+string(reset)+"\n", logStr)
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", logStr)
	}
}

func CDebug(format string, args ...interface{}) {
	CPrintfln(NORMAL, format, args...)
}

func CInfo(format string, args ...interface{}) {
	CPrintfln(NGreen, format, args...)
}

func CWarn(format string, args ...interface{}) {
	CEPrintfln(BMagenta, format, args...)
}

func CError(format string, args ...interface{}) {
	CEPrintfln(NRed, format, args...)
}

func CFatal(format string, args ...interface{}) {
	CEPrintfln(BRed, format, args...)
}
