package dubbo3

import "github.com/golang/protobuf/proto"

type CodeC interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type ProtobufCodeC struct {
}

func (p *ProtobufCodeC) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}
func (p *ProtobufCodeC) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}

func NewProtobufCodeC() CodeC {
	return &ProtobufCodeC{}
}
