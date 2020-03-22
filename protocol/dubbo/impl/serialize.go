package impl

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
)

type Serializer interface {
	Marshal(p DubboPackage) ([]byte, error)
	Unmarshal([]byte, *DubboPackage) error
}

func LoadSerializer(p *DubboPackage) error {
	// NOTE: default serialID is S_Hessian
	serialID := p.Header.SerialID
	if serialID == 0 {
		serialID = constant.S_Hessian2
	}
	serializer, err := extension.GetSerializerById(serialID)
	if err != nil {
		return err
	}
	p.SetSerializer(serializer.(Serializer))
	return nil
}
