package protocol

import (
	"sync"
)

import (
	log "github.com/AlexStocks/log4go"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/common"
)

// Extension - protocol
type Protocol interface {
	Export(invoker Invoker) Exporter
	Refer(url common.URL) Invoker
	Destroy()
}

// wrapping invoker
type Exporter interface {
	GetInvoker() Invoker
	Unexport()
}

/////////////////////////////
// base protocol
/////////////////////////////

type BaseProtocol struct {
	exporterMap *sync.Map
	invokers    []Invoker
}

func NewBaseProtocol() BaseProtocol {
	return BaseProtocol{
		exporterMap: new(sync.Map),
	}
}

func (bp *BaseProtocol) SetExporterMap(key string, exporter Exporter) {
	bp.exporterMap.Store(key, exporter)
}

func (bp *BaseProtocol) ExporterMap() *sync.Map {
	return bp.exporterMap
}

func (bp *BaseProtocol) SetInvokers(invoker Invoker) {
	bp.invokers = append(bp.invokers, invoker)
}

func (bp *BaseProtocol) Invokers() []Invoker {
	return bp.invokers
}

func (bp *BaseProtocol) Export(invoker Invoker) Exporter {
	return nil
}

func (bp *BaseProtocol) Refer(url common.URL) Invoker {
	return nil
}

// Destroy will destroy all invoker and exporter, so it only is called once.
func (bp *BaseProtocol) Destroy() {
	// destroy invokers
	for _, invoker := range bp.invokers {
		if invoker != nil {
			invoker.Destroy()
		}
	}
	bp.invokers = []Invoker{}

	// unexport exporters
	bp.exporterMap.Range(func(key, exporter interface{}) bool {
		if exporter != nil {
			exporter.(Exporter).Unexport()
		} else {
			bp.exporterMap.Delete(key)
		}
		return true
	})
}

/////////////////////////////
// base exporter
/////////////////////////////

type BaseExporter struct {
	key         string
	invoker     Invoker
	exporterMap *sync.Map
}

func NewBaseExporter(key string, invoker Invoker, exporterMap *sync.Map) *BaseExporter {
	return &BaseExporter{
		key:         key,
		invoker:     invoker,
		exporterMap: exporterMap,
	}
}

func (de *BaseExporter) GetInvoker() Invoker {
	return de.invoker

}

func (de *BaseExporter) Unexport() {
	log.Info("Exporter unexport.")
	de.invoker.Destroy()
	de.exporterMap.Delete(de.key)
}
