package protocol

import (
	"context"
	"testing"
)
import (
	"github.com/stretchr/testify/assert"
)
import (
	cluster "github.com/dubbo/go-for-apache-dubbo/cluster/support"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/config"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	"github.com/dubbo/go-for-apache-dubbo/protocol/protocolwrapper"
	"github.com/dubbo/go-for-apache-dubbo/registry"
)

func referNormal(t *testing.T, regProtocol *RegistryProtocol) {
	extension.SetProtocol("registry", GetProtocol)
	extension.SetRegistry("mock", registry.NewMockRegistry)
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)
	extension.SetCluster("mock", cluster.NewMockCluster)

	url, _ := config.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl, _ := config.NewURL(context.TODO(), "dubbo://127.0.0.1:20000//", config.WithParamsValue(constant.CLUSTER_KEY, "mock"))

	url.SubURL = &suburl

	invoker := regProtocol.Refer(url)
	assert.IsType(t, &protocol.BaseInvoker{}, invoker)
	assert.Equal(t, invoker.GetUrl().String(), url.String())
}
func TestRefer(t *testing.T) {
	regProtocol := NewRegistryProtocol()
	referNormal(t, regProtocol)
}

func TestMultiRegRefer(t *testing.T) {
	regProtocol := NewRegistryProtocol()
	referNormal(t, regProtocol)
	url2, _ := config.NewURL(context.TODO(), "mock://127.0.0.1:2222")
	suburl2, _ := config.NewURL(context.TODO(), "dubbo://127.0.0.1:20000//", config.WithParamsValue(constant.CLUSTER_KEY, "mock"))

	url2.SubURL = &suburl2

	regProtocol.Refer(url2)
	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 2)
}

func TestOneRegRefer(t *testing.T) {
	regProtocol := NewRegistryProtocol()
	referNormal(t, regProtocol)

	url2, _ := config.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl2, _ := config.NewURL(context.TODO(), "dubbo://127.0.0.1:20000//", config.WithParamsValue(constant.CLUSTER_KEY, "mock"))

	url2.SubURL = &suburl2

	regProtocol.Refer(url2)
	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 1)
}
func exporterNormal(t *testing.T, regProtocol *RegistryProtocol) {
	extension.SetProtocol("registry", GetProtocol)
	extension.SetRegistry("mock", registry.NewMockRegistry)
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)
	url, _ := config.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl, _ := config.NewURL(context.TODO(), "dubbo://127.0.0.1:20000//", config.WithParamsValue(constant.CLUSTER_KEY, "mock"))

	url.SubURL = &suburl
	invoker := protocol.NewBaseInvoker(url)
	exporter := regProtocol.Export(invoker)

	assert.IsType(t, &protocol.BaseExporter{}, exporter)
	assert.Equal(t, exporter.GetInvoker().GetUrl().String(), suburl.String())
}

func TestExporter(t *testing.T) {
	regProtocol := NewRegistryProtocol()
	exporterNormal(t, regProtocol)
}

func TestMultiRegAndMultiProtoExporter(t *testing.T) {
	regProtocol := NewRegistryProtocol()
	exporterNormal(t, regProtocol)

	url2, _ := config.NewURL(context.TODO(), "mock://127.0.0.1:2222")
	suburl2, _ := config.NewURL(context.TODO(), "jsonrpc://127.0.0.1:20000//", config.WithParamsValue(constant.CLUSTER_KEY, "mock"))

	url2.SubURL = &suburl2
	invoker2 := protocol.NewBaseInvoker(url2)
	regProtocol.Export(invoker2)

	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 2)

	var count2 int
	regProtocol.bounds.Range(func(key, value interface{}) bool {
		count2++
		return true
	})
	assert.Equal(t, count2, 2)
}

func TestOneRegAndProtoExporter(t *testing.T) {
	regProtocol := NewRegistryProtocol()
	exporterNormal(t, regProtocol)

	url2, _ := config.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl2, _ := config.NewURL(context.TODO(), "dubbo://127.0.0.1:20000//", config.WithParamsValue(constant.CLUSTER_KEY, "mock"))

	url2.SubURL = &suburl2
	invoker2 := protocol.NewBaseInvoker(url2)
	regProtocol.Export(invoker2)

	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 1)

	var count2 int
	regProtocol.bounds.Range(func(key, value interface{}) bool {
		count2++
		return true
	})
	assert.Equal(t, count2, 1)
}

func TestDestry(t *testing.T) {
	regProtocol := NewRegistryProtocol()
	referNormal(t, regProtocol)
	exporterNormal(t, regProtocol)

	regProtocol.Destroy()
	assert.Equal(t, len(regProtocol.invokers), 0)

	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 0)

	var count2 int
	regProtocol.bounds.Range(func(key, value interface{}) bool {
		count2++
		return true
	})
	assert.Equal(t, count2, 0)
}
