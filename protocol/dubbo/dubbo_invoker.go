package dubbo

import "github.com/dubbo/dubbo-go/config"

type DubboInvoker struct {
	url config.ConfigURL
}

func (di *DubboInvoker) Invoke() {

}

func (di *DubboInvoker) GetURL() config.ConfigURL {
	return di.url
}

func (di *DubboInvoker) Destroy() {

}
