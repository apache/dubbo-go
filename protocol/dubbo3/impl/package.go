package impl

import (
	"encoding/binary"
	"github.com/apache/dubbo-go/remoting"
)

type TripleRequestHeader struct {
}

type TriplePackageHandler struct {
	header *TripleHeader
	//codec  *CodeC
}

func (t *TriplePackageHandler) Frame2PkgData(frameData []byte) []byte {
	lineHeader := frameData[:5]
	length := binary.BigEndian.Uint32(lineHeader[1:])
	return frameData[5 : 5+length]
}
func (t *TriplePackageHandler) Pkg2FrameData(pkgData []byte) []byte {
	// todo 放到实现里面
	rsp := make([]byte, 5+len(pkgData))
	rsp[0] = byte(0)
	binary.BigEndian.PutUint32(rsp[1:], uint32(len(pkgData)))
	copy(rsp[5:], pkgData[:])
	return rsp
}

func init() {
	remoting.SetPackageHandler("dubbo3", NewTriplePkgHandler)
}

func NewTriplePkgHandler() remoting.PackageHandler {
	return &TriplePackageHandler{}
}
