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
