package dubbo

type RequestPayload struct {
	Params      interface{}
	Attachments map[string]string
}

func NewRequestPayload(args interface{}, atta map[string]string) *RequestPayload{
	if atta == nil {
		atta = make(map[string]string)
	}
	return &RequestPayload{
		Params:      args,
		Attachments: atta,
	}
}

func EnsureRequestPayload(body interface{}) *RequestPayload {
	if req, ok := body.(*RequestPayload); ok {
		return req
	}
	return NewRequestPayload(body, nil)
}
