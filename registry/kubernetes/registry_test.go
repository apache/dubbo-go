package kubernetes

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
	"net/url"
	"strconv"
	"strings"

	perrors "github.com/pkg/errors"
)

type demo struct{
	ctx  context.Context
}

func (d *demo) GetUrl() common.URL {


}

func (d *demo) IsAvailable() bool {
	return true
}

func (d *demo) Destroy() {


}

func (r *demo) Register(svc common.URL) error {

	role, err := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	if err != nil {
		return perrors.WithMessage(err, "get registry role")
	}

	r.cltLock.Lock()
	if _, ok := r.services[svc.Key()]; ok {
		r.cltLock.Unlock()
		return perrors.New(fmt.Sprintf("Path{%s} has been registered", svc.Path))
	}
	r.cltLock.Unlock()

	switch role {
	case common.PROVIDER:
		logger.Debugf("(provider register )Register(conf{%#v})", svc)
		if err := r.registerProvider(svc); err != nil {
			return perrors.WithMessage(err, "register provider")
		}
	case common.CONSUMER:
		logger.Debugf("(consumer register )Register(conf{%#v})", svc)
		if err := r.registerConsumer(svc); err != nil {
			return perrors.WithMessage(err, "register consumer")
		}
	default:
		return perrors.New(fmt.Sprintf("unknown role %d", role))
	}

	r.cltLock.Lock()
	r.services[svc.Key()] = svc
	r.cltLock.Unlock()
	return nil
}

func (*demo) Subscribe(*common.URL, registry.NotifyListener) {
	panic("implement me")
}

func (r *demo) registerProvider(svc common.URL) error {

	if len(svc.Path) == 0 || len(svc.Methods) == 0 {
		return perrors.New(fmt.Sprintf("service path %s or service method %s", svc.Path, svc.Methods))
	}

	var (
		urlPath    string
		encodedURL string
		dubboPath  string
	)

	providersNode := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), common.DubboNodes[common.PROVIDER])
	if err := r.createDirIfNotExist(providersNode); err != nil {
		return perrors.WithMessage(err, "create provider node")
	}

	params := url.Values{}

	svc.RangeParams(func(key, value string) bool {
		params[key] = []string{value}
		return true
	})
	params.Add("pid", processID)
	params.Add("ip", localIP)
	params.Add("anyhost", "true")
	params.Add("category", (common.RoleType(common.PROVIDER)).String())
	params.Add("dubbo", "dubbo-provider-golang-"+constant.Version)
	params.Add("side", (common.RoleType(common.PROVIDER)).Role())

	if len(svc.Methods) == 0 {
		params.Add("methods", strings.Join(svc.Methods, ","))
	}

	logger.Debugf("provider url params:%#v", params)
	var host string
	if len(svc.Ip) == 0 {
		host = localIP + ":" + svc.Port
	} else {
		host = svc.Ip + ":" + svc.Port
	}

	urlPath = svc.Path

	encodedURL = url.QueryEscape(fmt.Sprintf("%s://%s%s?%s", svc.Protocol, host, urlPath, params.Encode()))
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", svc.Service(), (common.RoleType(common.PROVIDER)).String())

	if err := r.client.Create(path.Join(dubboPath, encodedURL), ""); err != nil {
		return perrors.WithMessagef(err, "create k/v in etcd (path:%s, url:%s)", dubboPath, encodedURL)
	}

	return nil
}

