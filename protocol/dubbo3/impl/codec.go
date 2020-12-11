package impl

import (
	"github.com/apache/dubbo-go/remoting"
	"github.com/golang/protobuf/proto"
)

func init() {
	remoting.SetDubbo3Serializer("protobuf", NewDubbo3CodeC)
}

type ProtobufCodeC struct {
}

func (p *ProtobufCodeC) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}
func (p *ProtobufCodeC) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}

func NewDubbo3CodeC() remoting.Dubbo3Serializer {
	return &ProtobufCodeC{}
}
