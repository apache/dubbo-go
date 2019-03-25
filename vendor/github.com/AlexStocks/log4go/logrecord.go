/******************************************************
# DESC    :
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2018-03-27 14:48
# FILE    : logrecord.go
******************************************************/

package log4go

import (
	"bytes"
	"encoding/json"
	"strconv"
	"time"
)

import (
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

/****** LogRecord ******/

// A LogRecord contains all of the pertinent information for each message
type LogRecord struct {
	Level   Level     `json:"level,omitempty"`     // The log level
	Created time.Time `json:"timestamp,omitempty"` // The time at which the log message was created (nanoseconds)
	Source  string    `json:"source,omitempty"`    // The message source
	Message string    `json:"log,omitempty"`       // The log message
}

// 不要尝试使用json.Marshal，这个函数使用了反射，效率很低，详细结果见测试
// func (r LogRecord) MarshalJSON() ([]byte, error) {
func (r LogRecord) JSON() []byte {
	var buf bytes.Buffer

	buf.WriteString("{\"level\":")
	buf.WriteString(strconv.Itoa(int(r.Level)))
	//buf.WriteString(",")
	//
	//buf.WriteString("\"timestamp\":")
	buf.WriteString(",\"timestamp\":")
	timeJson, _ := r.Created.MarshalJSON()
	buf.Write(timeJson)
	//buf.WriteString(",")
	//
	//buf.WriteString("\"source\":")
	//buf.WriteString("\"")

	buf.WriteString(",\"source\":\"")
	buf.WriteString(r.Source)
	//buf.WriteString("\"")
	//buf.WriteString(",")
	//
	//buf.WriteString("\"log\":")
	//buf.WriteString("\"")

	buf.WriteString("\",\"log\":\"")
	buf.WriteString(r.Message)
	//buf.WriteString("\"")
	//buf.WriteString("}")
	buf.WriteString("\"}")

	return buf.Bytes()
}

func easyjson15d5d517Decodeg(in *jlexer.Lexer, out *LogRecord) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "level":
			out.Level = Level(in.Int())
		case "timestamp":
			if data := in.Raw(); in.Ok() {
				in.AddError((out.Created).UnmarshalJSON(data))
			}
		case "source":
			out.Source = string(in.String())
		case "log":
			out.Message = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson15d5d517Encodeg(out *jwriter.Writer, in LogRecord) {
	out.RawByte('{')
	first := true
	_ = first
	if in.Level != 0 {
		const prefix string = ",\"level\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int(int(in.Level))
	}
	if true {
		const prefix string = ",\"timestamp\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Raw((in.Created).MarshalJSON())
	}
	if in.Source != "" {
		const prefix string = ",\"source\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Source))
	}
	if in.Message != "" {
		const prefix string = ",\"log\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Message))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v LogRecord) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson15d5d517Encodeg(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v LogRecord) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson15d5d517Encodeg(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *LogRecord) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson15d5d517Decodeg(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *LogRecord) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson15d5d517Decodeg(l, v)
}
