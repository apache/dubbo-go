// Copyright (c) 2016 ~ 2018, Alex Stocks.
// Copyright (c) 2015 Alex Efros.
//
// The MIT License (MIT)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

import (
	jerrors "github.com/juju/errors"
)

const (
	MAX_JSONRPC_ID = 0x7FFFFFFF
	VERSION        = "2.0"
)

//////////////////////////////////////////
// codec type
//////////////////////////////////////////

type CodecType int

const (
	CODECTYPE_UNKNOWN CodecType = iota
	CODECTYPE_JSONRPC
)

var codecTypeStrings = [...]string{
	"unknown",
	"jsonrpc",
}

func (c CodecType) String() string {
	typ := CODECTYPE_UNKNOWN
	switch c {
	case CODECTYPE_JSONRPC:
		typ = c
	}

	return codecTypeStrings[typ]
}

func GetCodecType(t string) CodecType {
	var typ = CODECTYPE_UNKNOWN

	switch t {
	case codecTypeStrings[CODECTYPE_JSONRPC]:
		typ = CODECTYPE_JSONRPC
	}

	return typ
}

//////////////////////////////////////////
// json codec
//////////////////////////////////////////

type CodecData struct {
	ID     int64
	Method string
	Args   interface{}
	Error  string
}

// response Error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *Error) Error() string {
	buf, err := json.Marshal(e)
	if err != nil {
		msg, err := json.Marshal(err.Error())
		if err != nil {
			msg = []byte("jsonrpc2.Error: json.Marshal failed")
		}
		return fmt.Sprintf(`{"code":%d,"message":%s}`, -32001, string(msg))
	}
	return string(buf)
}

type clientRequest struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int64       `json:"id"`
}

type clientResponse struct {
	Version string           `json:"jsonrpc"`
	ID      int64            `json:"id"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *Error           `json:"error,omitempty"`
}

func (r *clientResponse) reset() {
	r.Version = ""
	r.ID = 0
	r.Result = nil
}

type jsonClientCodec struct {
	// temporary work space
	req clientRequest
	rsp clientResponse

	pending map[int64]string
}

func newJsonClientCodec() *jsonClientCodec {
	return &jsonClientCodec{
		pending: make(map[int64]string),
	}
}

func (c *jsonClientCodec) Write(d *CodecData) ([]byte, error) {
	// If return error: it will be returned as is for this call.
	// Allow param to be only Array, Slice, Map or Struct.
	// When param is nil or uninitialized Map or Slice - omit "params".
	param := d.Args
	if param != nil {
		switch k := reflect.TypeOf(param).Kind(); k {
		case reflect.Map:
			if reflect.TypeOf(param).Key().Kind() == reflect.String {
				if reflect.ValueOf(param).IsNil() {
					param = nil
				}
			}
		case reflect.Slice:
			if reflect.ValueOf(param).IsNil() {
				param = nil
			}
		case reflect.Array, reflect.Struct:
		case reflect.Ptr:
			switch k := reflect.TypeOf(param).Elem().Kind(); k {
			case reflect.Map:
				if reflect.TypeOf(param).Elem().Key().Kind() == reflect.String {
					if reflect.ValueOf(param).Elem().IsNil() {
						param = nil
					}
				}
			case reflect.Slice:
				if reflect.ValueOf(param).Elem().IsNil() {
					param = nil
				}
			case reflect.Array, reflect.Struct:
			default:
				return nil, jerrors.New("unsupported param type: Ptr to " + k.String())
			}
		default:
			return nil, jerrors.New("unsupported param type: " + k.String())
		}
	}

	c.req.Version = "2.0"
	c.req.Method = d.Method
	c.req.Params = param
	c.req.ID = d.ID & MAX_JSONRPC_ID
	// can not use d.ID. otherwise you will get error: can not find method of rsponse id 280698512
	c.pending[c.req.ID] = d.Method

	buf := bytes.NewBuffer(nil)
	defer buf.Reset()
	enc := json.NewEncoder(buf)
	if err := enc.Encode(&c.req); err != nil {
		return nil, jerrors.Trace(err)
	}

	return buf.Bytes(), nil
}

func (c *jsonClientCodec) Read(streamBytes []byte, x interface{}) error {
	c.rsp.reset()

	buf := bytes.NewBuffer(streamBytes)
	defer buf.Reset()
	dec := json.NewDecoder(buf)
	if err := dec.Decode(&c.rsp); err != nil {
		if err != io.EOF {
			err = jerrors.Trace(err)
		}
		return err
	}

	_, ok := c.pending[c.rsp.ID]
	if !ok {
		err := jerrors.Errorf("can not find method of rsponse id %v, rsponse error:%v", c.rsp.ID, c.rsp.Error)
		return err
	}
	delete(c.pending, c.rsp.ID)

	// c.rsp.ID
	if c.rsp.Error != nil {
		return jerrors.New(c.rsp.Error.Error())
	}

	return jerrors.Trace(json.Unmarshal(*c.rsp.Result, x))
}
