package cluster

import (
	gxnet "github.com/AlexStocks/goext/net"
	jerrors "github.com/juju/errors"
	"github.com/tevino/abool"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster"
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	"github.com/dubbo/go-for-apache-dubbo/version"
)

type baseClusterInvoker struct {
	directory      cluster.Directory
	availablecheck bool
	destroyed      *abool.AtomicBool
}

func newBaseClusterInvoker(directory cluster.Directory) baseClusterInvoker {
	return baseClusterInvoker{
		directory:      directory,
		availablecheck: true,
		destroyed:      abool.NewBool(false),
	}
}
func (invoker *baseClusterInvoker) GetUrl() common.URL {
	return invoker.directory.GetUrl()
}

func (invoker *baseClusterInvoker) Destroy() {
	//this is must atom operation
	if invoker.destroyed.SetToIf(false, true) {
		invoker.directory.Destroy()
	}
}

func (invoker *baseClusterInvoker) IsAvailable() bool {
	//TODO:不理解java版本中关于stikyInvoker的逻辑所以先不写
	return invoker.directory.IsAvailable()
}

//check invokers availables
func (invoker *baseClusterInvoker) checkInvokers(invokers []protocol.Invoker, invocation protocol.Invocation) error {
	if len(invokers) == 0 {
		ip, _ := gxnet.GetLocalIP()
		return jerrors.Errorf("Failed to invoke the method %v . No provider available for the service %v from"+
			"registry %v on the consumer %v using the dubbo version %v .Please check if the providers have been started and registered.",
			invocation.MethodName(), invoker.directory.GetUrl().Key(), invoker.directory.GetUrl().String(), ip, version.Version)
	}
	return nil

}

//check cluster invoker is destroyed or not
func (invoker *baseClusterInvoker) checkWhetherDestroyed() error {
	if invoker.destroyed.IsSet() {
		ip, _ := gxnet.GetLocalIP()
		return jerrors.Errorf("Rpc cluster invoker for %v on consumer %v use dubbo version %v is now destroyed! can not invoke any more. ",
			invoker.directory.GetUrl().Service(), ip, version.Version)
	}
	return nil
}

func (invoker *baseClusterInvoker) doSelect(lb cluster.LoadBalance, invocation protocol.Invocation, invokers []protocol.Invoker, invoked []protocol.Invoker) protocol.Invoker {
	//todo:ticky connect 粘纸连接
	if len(invokers) == 1 {
		return invokers[0]
	}
	selectedInvoker := lb.Select(invokers, invoker.GetUrl(), invocation)

	//judge to if the selectedInvoker is invoked

	if !selectedInvoker.IsAvailable() || !invoker.availablecheck || isInvoked(selectedInvoker, invoked) {
		// do reselect
		var reslectInvokers []protocol.Invoker

		for _, invoker := range invokers {
			if !invoker.IsAvailable() {
				continue
			}

			if !isInvoked(invoker, invoked) {
				reslectInvokers = append(reslectInvokers, invoker)
			}
		}

		if len(reslectInvokers) > 0 {
			return lb.Select(reslectInvokers, invoker.GetUrl(), invocation)
		} else {
			return nil
		}
	}
	return selectedInvoker

}

func isInvoked(selectedInvoker protocol.Invoker, invoked []protocol.Invoker) bool {
	if len(invoked) > 0 {
		for _, i := range invoked {
			if i == selectedInvoker {
				return true
			}
		}
	}
	return false
}
