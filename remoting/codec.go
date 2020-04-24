package remoting

import (
	"bytes"
)

type Codec interface {
	EncodeRequest(request *Request) (*bytes.Buffer, error)
	EncodeResponse(response *Response) (*bytes.Buffer, error)
	DecodeRequest(*bytes.Buffer) (*Request, int, error)
	DecodeResponse(*bytes.Buffer) (*Response, int, error)
}

var (
	codec map[string]*Codec
)

func init() {
	codec = make(map[string]*Codec, 2)
}

func NewCodec(protocol string, codecTmp *Codec) {
	codec[protocol] = codecTmp
}

func GetCodec(protocol string) *Codec {
	return codec[protocol]
}
