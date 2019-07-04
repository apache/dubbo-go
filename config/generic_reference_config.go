package config

import (
	"context"
	"github.com/apache/dubbo-go/common/logger"
)

type GenericService struct {
	Invoke       func(req []interface{}) (interface{}, error) `dubbo:"$invoke"`
	referenceStr string
}

func NewGenericService(reference string) *GenericService {
	return &GenericService{referenceStr: reference}
}

func (u *GenericService) Reference() string {
	return u.referenceStr
}

type GenericConsumerConfig struct {
	Protocol       string
	Registry       string
	Version        string
	Group          string
	InterfaceName  string
	Cluster        string
	Retries        int64
	ref            *ReferenceConfig
	genericService *GenericService
}

func (gConfig *GenericConsumerConfig) LoadGenericReferenceConfig(key string) {
	gConfig.genericService = NewGenericService(key)
	SetConsumerService(gConfig.genericService)
	gConfig.NewGenericReferenceConfig(key)

	rpcService := GetConsumerService(key)
	if rpcService == nil {
		logger.Warnf("%s is not exsist!", key)
		return
	}

	gConfig.ref.id = key
	gConfig.ref.Refer()
	gConfig.ref.Implement(rpcService)

}
func (gConfig *GenericConsumerConfig) GetService() *GenericService {
	return gConfig.genericService
}
func (gConfig *GenericConsumerConfig) NewGenericReferenceConfig(id string) {
	gr := NewReferenceConfig(id, context.TODO())
	//gr.Filter = "genericConsumer" //todo: add genericConsumer filter
	gr.Registry = gConfig.Registry
	gr.Protocol = gConfig.Protocol
	gr.Version = gConfig.Version
	gr.Group = gConfig.Group
	gr.InterfaceName = gConfig.InterfaceName
	gr.Cluster = gConfig.Cluster
	gr.Methods = append(gr.Methods, &MethodConfig{Name: "$invoke", Retries: gConfig.Retries})
	gConfig.ref = gr
}
