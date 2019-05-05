package registry

import "github.com/dubbo/dubbo-go/config"

type MockRegistry struct {
}

func (*MockRegistry) Register(url config.URL) error {
	return nil
}

func (*MockRegistry) Close() {

}
func (*MockRegistry) IsClosed() bool {
	return false
}

//func (*MockRegistry) Subscribe(config.URL) (Listener, error) {
//
//}
//
//type listener struct{}
//
//func Next() (*ServiceEvent, error) {
//
//}
