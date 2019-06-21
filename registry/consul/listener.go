package consul

import (
	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/watch"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/registry"
)

type consulListener struct {
	plan      *watch.Plan
	addrsChan chan []*consul.ServiceEntry
}

func newConsulListener(url common.URL) (registry.Listener, error) {
	var err error

	addrsChan := make(chan []*consul.ServiceEntry, 1)

	params := make(map[string]interface{})
	params["type"] = "service"
	params["service"] = url.Service()
	plan, err := watch.Parse(params)
	if err != nil {
		return nil, err
	}
	plan.Handler = func(idx uint64, raw interface{}) {
		addrs, _ := raw.([]*consul.ServiceEntry)
		addrsChan <- addrs
	}

	listener := &consulListener{
		plan:      plan,
		addrsChan: addrsChan,
	}
	return listener, nil
}

func (l *consulListener) Next() (*registry.ServiceEvent, error) {
	return nil, nil
}

func (l *consulListener) Close() {

}