/******************************************************
# DESC    : output log with color
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2018-03-25 17:51
# FILE    : color.go
******************************************************/

package log4go

import (
	"fmt"
	"io"
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

func CPrintfln(w io.Writer, color []byte, logString string) {
	if f, ok := w.(*os.File); ok {
		if isatty.IsTerminal(f.Fd()) {
			fmt.Fprintf(f, string(color)+"%s"+string(reset), logString)
			return
		}
	}

	fmt.Fprintf(w, fmt.Sprintf("%s", logString))
}

func cDebug(w io.Writer, logString string) {
	CPrintfln(w, NORMAL, logString)
}

func cTrace(w io.Writer, logString string) {
	CPrintfln(w, NBlue, logString)
}

func cInfo(w io.Writer, logString string) {
	CPrintfln(w, NGreen, logString)
}

func cWarn(w io.Writer, logString string) {
	CPrintfln(w, BMagenta, logString)
}

func cError(w io.Writer, logString string) {
	CPrintfln(w, NRed, logString)
}

func cCritical(w io.Writer, logString string) {
	CPrintfln(w, BRed, logString)
}
