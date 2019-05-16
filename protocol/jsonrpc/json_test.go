package jsonrpc

import (
	"encoding/json"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

type TestData struct {
	Test string
}

func TestJsonClientCodec_Write(t *testing.T) {
	cd := &CodecData{
		ID:     1,
		Method: "GetUser",
		Args:   []interface{}{"args", 2},
	}
	codec := newJsonClientCodec()
	data, err := codec.Write(cd)
	assert.NoError(t, err)
	assert.Equal(t, "{\"jsonrpc\":\"2.0\",\"method\":\"GetUser\",\"params\":[\"args\",2],\"id\":1}\n", string(data))
}

func TestJsonClientCodec_Read(t *testing.T) {
	codec := newJsonClientCodec()
	codec.pending[1] = "GetUser"
	rsp := &TestData{}
	err := codec.Read([]byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"Test\":\"test\"}}\n"), rsp)
	assert.NoError(t, err)
	assert.Equal(t, "test", rsp.Test)

	//error
	codec.pending[1] = "GetUser"
	err = codec.Read([]byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"error\":{\"code\":-32000,\"message\":\"error\"}}\n"), rsp)
	assert.EqualError(t, err, "{\"code\":-32000,\"message\":\"error\"}")
}

func TestServerCodec_Write(t *testing.T) {
	codec := newServerCodec()
	a := json.RawMessage([]byte("1"))
	codec.req = serverRequest{Version: "1.0", Method: "GetUser", ID: &a}
	data, err := codec.Write("error", &TestData{Test: "test"})
	assert.NoError(t, err)
	assert.Equal(t, "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"Test\":\"test\"},\"error\":{\"code\":-32000,\"message\":\"error\"}}\n", string(data))
}

func TestServerCodec_Read(t *testing.T) {
	codec := newServerCodec()
	header := map[string]string{}
	err := codec.ReadHeader(header, []byte("{\"jsonrpc\":\"2.0\",\"method\":\"GetUser\",\"params\":[\"args\",2],\"id\":1}\n"))
	assert.EqualError(t, err, "{\"code\":-32601,\"message\":\"Method not found\"}")

	header["HttpMethod"] = "POST"
	err = codec.ReadHeader(header, []byte("{\"jsonrpc\":\"2.0\",\"method\":\"GetUser\",\"params\":[\"args\",2],\"id\":1}\n"))
	assert.NoError(t, err)
	assert.Equal(t, "1", string([]byte(*codec.req.ID)))
	assert.Equal(t, "GetUser", codec.req.Method)
	assert.Equal(t, "2.0", codec.req.Version)
	assert.Equal(t, "[\"args\",2]", string([]byte(*codec.req.Params)))

	req := []interface{}{}
	err = codec.ReadBody(&req)
	assert.NoError(t, err)
	assert.Equal(t, "args", req[0])
	assert.Equal(t, float64(2), req[1])
}
