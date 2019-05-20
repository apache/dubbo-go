package config

import "context"

type MockService struct {
}

func (*MockService) Service() string {
	return "MockService"
}
func (*MockService) Version() string {
	return "1.0"
}
func (*MockService) GetUser(ctx context.Context, itf []interface{}, str *struct{}) error {
	return nil
}
func (*MockService) GetUser1(ctx context.Context, itf []interface{}, str *struct{}) error {
	return nil
}
