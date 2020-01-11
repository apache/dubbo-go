package rest

import "github.com/apache/dubbo-go/protocol"

type RestExporter struct {
	protocol.BaseExporter
}

func NewRestExporter() *RestExporter {
	return &RestExporter{}
}

func (re *RestExporter) Unexport() {
	// undeploy serviceßß
	return
}
