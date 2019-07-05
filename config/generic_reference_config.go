package config

type GenericService struct {
	Invoke       func(req []interface{}) (interface{}, error) `dubbo:"$invoke"`
	referenceStr string
}

func NewGenericService(referenceStr string) *GenericService {
	return &GenericService{referenceStr: referenceStr}
}

func (u *GenericService) Reference() string {
	return u.referenceStr
}

func (refconfig *ReferenceConfig) Load(id string) {
	//gr.Filter = "genericConsumer" //todo: add genericConsumer filter
	genericService := NewGenericService(refconfig.id)
	SetConsumerService(genericService)
	refconfig.id = id
	refconfig.Refer()
	refconfig.Implement(genericService)
	return
}
