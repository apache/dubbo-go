package config

import (
	"context"
	"errors"
)

const GenericReferKey = "GenericReferKey"

type GenericService struct {
	Invoke       func(req []interface{}) (interface{}, error) `dubbo:"$invoke"`
	referenceStr string
}

func NewGenericService(referenceStr string) *GenericService {
	return &GenericService{referenceStr: GenericReferKey}
}

func (u *GenericService) Reference() string {
	return u.referenceStr
}

type GenericConsumerConfig struct {
	ID             string
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

func (gConfig *GenericConsumerConfig) Load() (err error) {

	gr := NewReferenceConfig(context.TODO())
	//gr.Filter = "genericConsumer" //todo: add genericConsumer filter
	gr.Registry = gConfig.Registry
	gr.Protocol = gConfig.Protocol
	gr.Version = gConfig.Version
	gr.Group = gConfig.Group
	gr.InterfaceName = gConfig.InterfaceName
	gr.Cluster = gConfig.Cluster
	gr.Methods = append(gr.Methods, &MethodConfig{Name: "$invoke", Retries: gConfig.Retries})
	gConfig.ref = gr
	gConfig.genericService = NewGenericService(gConfig.ID)
	SetConsumerService(gConfig.genericService)
	rpcService := GetConsumerService(GenericReferKey)
	if rpcService == nil {
		err = errors.New("get rpcService err,GenericReferKey not Set ")
		return
	}

	gConfig.ref.id = gConfig.ID
	gConfig.ref.Refer()
	gConfig.ref.Implement(rpcService)
	return

}
func (gConfig *GenericConsumerConfig) GetGenericService() *GenericService {
	return gConfig.genericService
}
