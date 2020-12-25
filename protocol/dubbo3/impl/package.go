/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package impl

import (
	"encoding/binary"
)

import (
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
