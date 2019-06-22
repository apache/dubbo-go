/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jsonrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
)

import (
	perrors "github.com/pkg/errors"
)

const (
	MAX_JSONRPC_ID = 0x7FFFFFFF
	VERSION        = "2.0"
)

type CodecData struct {
	ID     int64
	Method string
	Args   interface{}
	Error  string
}

const (
	// Errors defined in the JSON-RPC spec. See
	// http://www.jsonrpc.org/specification#error_object.
	CodeParseError       = -32700
	CodeInvalidRequest   = -32600
	CodeMethodNotFound   = -32601
	CodeInvalidParams    = -32602
	CodeInternalError    = -32603
	codeServerErrorStart = -32099
	codeServerErrorEnd   = -32000
)

// rsponse Error
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

//////////////////////////////////////////
// json client codec
//////////////////////////////////////////

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
				return nil, perrors.New("unsupported param type: Ptr to " + k.String())
			}
		default:
			return nil, perrors.New("unsupported param type: " + k.String())
		}
	}

	c.req.Version = "2.0"
	c.req.Method = d.Method
	c.req.Params = param
	c.req.ID = d.ID & MAX_JSONRPC_ID
	// can not use d.ID. otherwise you will get error: can not find method of response id 280698512
	c.pending[c.req.ID] = d.Method

	buf := bytes.NewBuffer(nil)
	defer buf.Reset()
	enc := json.NewEncoder(buf)
	if err := enc.Encode(&c.req); err != nil {
		return nil, perrors.WithStack(err)
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
			err = perrors.WithStack(err)
		}
		return err
	}

	_, ok := c.pending[c.rsp.ID]
	if !ok {
		err := perrors.Errorf("can not find method of rsponse id %v, rsponse error:%v", c.rsp.ID, c.rsp.Error)
		return err
	}
	delete(c.pending, c.rsp.ID)

	// c.rsp.ID
	if c.rsp.Error != nil {
		return perrors.New(c.rsp.Error.Error())
	}

	if c.rsp.Result == nil {
		return nil
	}
	return perrors.WithStack(json.Unmarshal(*c.rsp.Result, x))
}

//////////////////////////////////////////
// json server codec
//////////////////////////////////////////

type serverRequest struct {
	Version string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	Params  *json.RawMessage `json:"params"`
	ID      *json.RawMessage `json:"id"`
}

func (r *serverRequest) reset() {
	r.Version = ""
	r.Method = ""
	if r.Params != nil {
		*r.Params = (*r.Params)[:0]
	}
	if r.ID != nil {
		*r.ID = (*r.ID)[:0]
	}
}

func (r *serverRequest) UnmarshalJSON(raw []byte) error {
	r.reset()

	type req *serverRequest
	// Attention: if do not define a new struct named @req, the json.Unmarshal will invoke
	// (*serverRequest)UnmarshalJSON recursively.
	if err := json.Unmarshal(raw, req(r)); err != nil {
		return perrors.New("bad request")
	}

	var o = make(map[string]*json.RawMessage)
	if err := json.Unmarshal(raw, &o); err != nil {
		return perrors.New("bad request")
	}
	if o["jsonrpc"] == nil || o["method"] == nil {
		return perrors.New("bad request")
	}
	_, okID := o["id"]
	_, okParams := o["params"]
	if len(o) == 3 && !(okID || okParams) || len(o) == 4 && !(okID && okParams) || len(o) > 4 {
		return perrors.New("bad request")
	}
	if r.Version != Version {
		return perrors.New("bad request")
	}
	if okParams {
		if r.Params == nil || len(*r.Params) == 0 {
			return perrors.New("bad request")
		}
		switch []byte(*r.Params)[0] {
		case '[', '{':
		default:
			return perrors.New("bad request")
		}
	}
	if okID && r.ID == nil {
		r.ID = &null
	}
	if okID {
		if len(*r.ID) == 0 {
			return perrors.New("bad request")
		}
		switch []byte(*r.ID)[0] {
		case 't', 'f', '{', '[':
			return perrors.New("bad request")
		}
	}

	return nil
}

type serverResponse struct {
	Version string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  interface{}      `json:"result,omitempty"`
	Error   interface{}      `json:"error,omitempty"`
}

type ServerCodec struct {
	req serverRequest
}

var (
	null    = json.RawMessage([]byte("null"))
	Version = "2.0"
)

func newServerCodec() *ServerCodec {
	return &ServerCodec{}
}

func (c *ServerCodec) ReadHeader(header map[string]string, body []byte) error {
	if header["HttpMethod"] != "POST" {
		return &Error{Code: -32601, Message: "Method not found"}
	}

	// If return error:
	// - codec will be closed
	// So, try to send error reply to client before returning error.

	buf := bytes.NewBuffer(body)
	defer buf.Reset()
	dec := json.NewDecoder(buf)

	var raw json.RawMessage
	c.req.reset()
	if err := dec.Decode(&raw); err != nil {
		// rspError := &Error{Code: -32700, Message: "Parse error"}
		// c.resp = serverResponse{Version: Version, ID: &null, Error: rspError}
		return err
	}
	if err := json.Unmarshal(raw, &c.req); err != nil {
		// if err.Error() == "bad request" {
		// 	rspError := &Error{Code: -32600, Message: "Invalid request"}
		// 	c.resp = serverResponse{Version: Version, ID: &null, Error: rspError}
		// }
		return err
	}

	return nil
}

func (c *ServerCodec) ReadBody(x interface{}) error {
	// If x!=nil and return error e:
	// - Write() will be called with e.Error() in r.Error
	if x == nil {
		return nil
	}
	if c.req.Params == nil {
		return nil
	}

	// 在这里把请求参数json 字符串转换成了相应的struct
	params := []byte(*c.req.Params)
	if err := json.Unmarshal(*c.req.Params, x); err != nil {
		// Note: if c.request.Params is nil it's not an error, it's an optional member.
		// JSON params structured object. Unmarshal to the args object.

		if 2 < len(params) && params[0] == '[' && params[len(params)-1] == ']' {
			// Clearly JSON params is not a structured object,
			// fallback and attempt an unmarshal with JSON params as
			// array value and RPC params is struct. Unmarshal into
			// array containing the request struct.
			params := [1]interface{}{x}
			if err = json.Unmarshal(*c.req.Params, &params); err != nil {
				return &Error{Code: -32602, Message: "Invalid params, error:" + err.Error()}
			}
		} else {
			return &Error{Code: -32602, Message: "Invalid params, error:" + err.Error()}
		}
	}

	return nil
}

func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

func newError(message string) *Error {
	switch {
	case strings.HasPrefix(message, "rpc: service/method request ill-formed"):
		return NewError(-32601, message)
	case strings.HasPrefix(message, "rpc: can't find service"):
		return NewError(-32601, message)
	case strings.HasPrefix(message, "rpc: can't find method"):
		return NewError(-32601, message)
	default:
		return NewError(-32000, message)
	}
}

func (c *ServerCodec) Write(errMsg string, x interface{}) ([]byte, error) {
	// If return error: nothing happens.
	// In r.Error will be "" or .Error() of error returned by:
	// - ReadBody()
	// - called RPC method
	resp := serverResponse{Version: Version, ID: c.req.ID, Result: x}
	if len(errMsg) == 0 {
		if x == nil {
			resp.Result = &null
		} else {
			resp.Result = x
		}
	} else if errMsg[0] == '{' && errMsg[len(errMsg)-1] == '}' {
		// Well& this check for '{'&'}' isn't too strict, but I
		// suppose we're trusting our own RPC methods (this way they
		// can force sending wrong reply or many replies instead
		// of one) and normal errors won't be formatted this way.
		raw := json.RawMessage(errMsg)
		resp.Error = &raw
	} else {
		raw := json.RawMessage(newError(errMsg).Error())
		resp.Error = &raw
	}

	buf := bytes.NewBuffer(nil)
	defer buf.Reset()
	enc := json.NewEncoder(buf)
	if err := enc.Encode(resp); err != nil {
		return nil, perrors.WithStack(err)
	}

	return buf.Bytes(), nil
}
