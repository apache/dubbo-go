package extension

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/pkg/errors"
)

var (
	serializers = make(map[string]interface{})
	nameMaps    = make(map[byte]string)
)

func init() {
	nameMaps = map[byte]string {
		constant.S_Hessian2: constant.HESSIAN2_SERIALIZATION,
		constant.S_Proto: constant.PROTOBUF_SERIALIZATION,
	}
}

func SetSerializer(name string, serializer interface{}) {
	serializers[name] = serializer
}

func GetSerializer(name string) interface{} {
	return serializers[name]
}

func GetSerializerById(id byte) (interface{}, error) {
	name, ok := nameMaps[id]
	if !ok {
		return nil, errors.Errorf("serialId %d not found", id)
	}
	serializer, ok := serializers[name]
	if !ok {
		return nil, errors.Errorf("serialization %s not found", name)
	}
	return serializer, nil
}

