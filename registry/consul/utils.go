package consul

import (
	"strconv"
	"crypto/md5"
)

import (
	consul "github.com/hashicorp/consul/api"
)

import (
	"github.com/apache/dubbo-go/common"
)

func buildId(url common.URL) string {
	return string(md5.Sum([]byte(url.String()))[:])
}

func buildService(url common.URL) (*consul.AgentServiceRegistration, error) {
	var err error

	// id
	id := buildId(url)

	// port
	port, err := strconv.Atoi(url.Port)
	if err != nil {
		return nil, err
	}

	// tags
	tags := make([]string, 0)
	for k := range url.Params {
		tags = append(tags, k + "=" + url.Params.Get(k))
	}

	// meta
	meta := make(map[string]string)
	meta["url"] = url.String()

	service := &consul.AgentServiceRegistration{
		Name:    url.Service(),
		ID:      id,
		Address: url.Ip,
		Port:    port,
		Tags:    tags,
		Meta:    meta,
	}

	return service, nil
}

func retrieveURL(service *consul.ServiceEntry) common.URL {
	url := service.Service.Meta["url"]
	return common.NewURLFromString(url)
}
