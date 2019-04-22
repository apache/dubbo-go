package dubbo

import "github.com/dubbo/dubbo-go/config"

type DubboInvoker struct {
	url config.URL
}

func (di *DubboInvoker) Invoke() {

}

func (di *DubboInvoker) GetURL() config.URL {
	return di.url
}

func (di *DubboInvoker) Destroy() {

}
