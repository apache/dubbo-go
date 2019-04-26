package protocol

import (
	"github.com/dubbo/dubbo-go/config"
	"github.com/prometheus/common/log"
)

// Extension - Protocol
type Protocol interface {
	Export(invoker Invoker) Exporter
	Refer(url config.IURL) Invoker
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
	exporterMap map[string]Exporter
	invokers    []Invoker
}

func NewBaseProtocol() BaseProtocol {
	return BaseProtocol{
		exporterMap: make(map[string]Exporter),
	}
}

func (bp *BaseProtocol) SetExporterMap(key string, exporter Exporter) {
	bp.exporterMap[key] = exporter
}

func (bp *BaseProtocol) ExporterMap() map[string]Exporter {
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

func (bp *BaseProtocol) Refer(url config.IURL) Invoker {
	return nil
}

func (bp *BaseProtocol) Destroy() {
	// destroy invokers
	for _, invoker := range bp.invokers {
		if invoker != nil {
			invoker.Destroy()
		}
	}
	bp.invokers = []Invoker{}

	// unexport exporters
	for key, exporter := range bp.ExporterMap() {
		if exporter != nil {
			exporter.Unexport()
		} else {
			delete(bp.exporterMap, key)
		}
	}
}

/////////////////////////////
// base exporter
/////////////////////////////

type BaseExporter struct {
	key         string
	invoker     Invoker
	exporterMap map[string]Exporter
}

func NewBaseExporter(key string, invoker Invoker, exporterMap map[string]Exporter) *BaseExporter {
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
	delete(de.exporterMap, de.key)
}
