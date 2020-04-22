package remoting

import (
	"bytes"
)

type Codec interface {
	EncodeRequest(request Request) (*bytes.Buffer, error)
	EncodeResponse(response Response) (*bytes.Buffer, error)
	DecodeRequest(*bytes.Buffer) (*Request, error)
	DecodeResponse(*bytes.Buffer) (*Response, error)
}
