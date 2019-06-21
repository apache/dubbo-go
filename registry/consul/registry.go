package consul

import (
	"strconv"
)

import (
	perrors "github.com/pkg/errors"
	consul "github.com/hashicorp/consul/api"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/common/constant"
)

type consulRegistry struct {
	*common.URL
	client *consul.Client
}

func newConsulRegistry(url *common.URL) (registry.Registry, error) {
	var err error

	config := &consul.Config{Address: url.Location}
	client, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}

	r := &consulRegistry{
		URL:    url,
		client: client,
	}

	return r, nil
}

func (r *consulRegistry) Register(url common.URL) error {
	var err error

	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	if role == common.PROVIDER {
		err = r.register(url)
		if err != nil {
			return perrors.WithStack(err)
		}
	}
	return nil
}

func (r *consulRegistry) register(url common.URL) error {
	var err error

	service, err := buildService(url)
	if err != nil {
		return err
	}
	return r.client.Agent().ServiceRegister(service)
}

func (r *consulRegistry) Subscribe(url common.URL) (registry.Listener, error) {
	var (
		listener registry.Listener
		err 	 error
	)

	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	if role == common.CONSUMER {
		listener, err = r.subscribe(url)
		if err != nil {
			return nil, err
		}
	}
	return listener, nil
}

func (r *consulRegistry) subscribe(url common.URL) (registry.Listener, error) {
	var err error

	listener, err := newConsulListener(url)
	return listener, err
}

func (r *consulRegistry) GetUrl() common.URL {
	return *r.URL
}

func (r *consulRegistry) IsAvailable() bool {

}

func (r *consulRegistry) Destroy() {

}
