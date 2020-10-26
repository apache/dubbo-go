package protocol

import (
	"sync"
)

type Protocol interface {
	Read([]byte, *sync.Map) (interface{}, int, error)
	Write(*Request) ([]byte, error)
}

type Request struct {
	ID          uint64
	InterfaceID string // interface寻址id
	Version     string
	Group       string
	Method      string
	Params      interface{}
	pojo        interface{}
}
