package impl

import "github.com/golang/protobuf/proto"

type ProtobufCodeC struct {
}

func (p *ProtobufCodeC) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}
func (p *ProtobufCodeC) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}

func NewDubbo3CodeC() *ProtobufCodeC {
	return &ProtobufCodeC{}
}
