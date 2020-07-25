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
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/apache/dubbo-go/common/logger"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/golang/protobuf/proto"
	"github.com/matttproud/golang_protobuf_extensions/pbutil"
	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	pb "github.com/apache/dubbo-go/protocol/dubbo/impl/proto"
)

type ProtoSerializer struct{}

func (p ProtoSerializer) Marshal(pkg DubboPackage) ([]byte, error) {
	if pkg.IsHeartBeat() {
		return []byte{byte('N')}, nil
	}
	if pkg.Body == nil {
		return nil, errors.New("package body should not be nil")
	}
	if pkg.IsRequest() {
		return marshalRequestProto(pkg)
	}
	return marshalResponseProto(pkg)
}

func (p ProtoSerializer) Unmarshal(data []byte, pkg *DubboPackage) error {
	if pkg.IsRequest() {
		return unmarshalRequestProto(data, pkg)
	}
	return unmarshalResponseProto(data, pkg)
}

func unmarshalResponseProto(data []byte, pkg *DubboPackage) error {
	if pkg.Body == nil {
		pkg.SetBody(NewResponsePayload(nil, nil, nil))
	}
	response := EnsureResponsePayload(pkg.Body)
	buf := bytes.NewBuffer(data)

	var responseType int32
	if err := readByte(buf, &responseType); err != nil {
		return err
	}

	hasAttachments := false
	hasException := false
	switch responseType {
	case RESPONSE_VALUE_WITH_ATTACHMENTS:
		hasAttachments = true
	case RESPONSE_WITH_EXCEPTION:
		hasException = true
	case RESPONSE_WITH_EXCEPTION_WITH_ATTACHMENTS:
		hasAttachments = true
		hasException = true
	}
	if hasException {
		throwable := pb.ThrowableProto{}
		if err := readObject(buf, &throwable); err != nil {
			return err
		}
		// generate error only with error message
		response.Exception = errors.New(throwable.OriginalMessage)
	} else {
		// read response body
		protoMsg, ok := response.RspObj.(proto.Message)
		if !ok {
			return errors.New("response rspobj not protobuf message")
		}
		if err := readObject(buf, protoMsg); err != nil {
			return err
		}
	}

	if hasAttachments {
		atta := pb.Map{}
		if err := readObject(buf, &atta); err != nil {
			return err
		}
		if response.Attachments == nil {
			response.Attachments = atta.Attachments
		} else {
			for k, v := range atta.Attachments {
				response.Attachments[k] = v
			}
		}

	}
	return nil
}

func unmarshalRequestProto(data []byte, pkg *DubboPackage) error {
	var dubboVersion string
	var svcPath string
	var svcVersion string
	var svcMethod string
	buf := bytes.NewBuffer(data)
	if err := readUTF(buf, &dubboVersion); err != nil {
		return err
	}
	if err := readUTF(buf, &svcPath); err != nil {
		return err
	}
	if err := readUTF(buf, &svcVersion); err != nil {
		return err
	}
	if err := readUTF(buf, &svcMethod); err != nil {
		return err
	}
	// NOTE: protobuf rpc just have exact one parameter, while golang doesn't need this field
	var argsType string
	if err := readUTF(buf, &argsType); err != nil {
		return err
	}
	// get raw body bytes for proxy methods to unmarshal
	var protoMsgLength int
	if err := readDelimitedLength(buf, &protoMsgLength); err != nil {
		return err
	}
	argBytes := make([]byte, protoMsgLength)
	if n, err := buf.Read(argBytes); err != nil {
		if n != protoMsgLength {
			return errors.New("illegal msg length")
		}
		return err
	}
	arg := getRegisterMessage(argsType)
	if !arg.IsZero() {
		err := proto.Unmarshal(argBytes, arg.Interface().(JavaProto))
		if err != nil {
			panic(err)
		}
	}

	m := &pb.Map{}
	if err := readObject(buf, m); err != nil {
		return err
	}
	svc := Service{}
	svc.Version = svcVersion
	svc.Method = svcMethod
	// just as hessian
	svc.Path = svcPath
	if svc.Path == "" && len(m.Attachments[constant.PATH_KEY]) > 0 {
		svc.Path = m.Attachments[constant.PATH_KEY]
	}

	if _, ok := m.Attachments[constant.INTERFACE_KEY]; ok {
		svc.Interface = m.Attachments[constant.INTERFACE_KEY]
	} else {
		svc.Interface = svc.Path
	}
	pkg.SetService(svc)
	pkg.SetBody(map[string]interface{}{
		"dubboVersion": dubboVersion,
		"args":         []interface{}{arg.Interface()},
		"service":      common.ServiceMap.GetService(DUBBO, svc.Path), // path as a key
		"attachments":  m.Attachments,
	})

	return nil
}

func marshalRequestProto(pkg DubboPackage) ([]byte, error) {
	request := EnsureRequestPayload(pkg.Body)
	args, ok := request.Params.([]interface{})
	buf := bytes.NewBuffer(make([]byte, 0))
	if !ok {
		return nil, errors.New("proto buffer args should be marshaled in []byte")
	}
	// NOTE: protobuf rpc just has exact one parameter
	if len(args) != 1 {
		return nil, errors.New("illegal protobuf service, len(arg) should equal 1")
	}
	// dubbo version
	if err := writeUTF(buf, DUBBO_VERSION); err != nil {
		return nil, err
	}
	// service path
	if err := writeUTF(buf, pkg.Service.Path); err != nil {
		return nil, err
	}
	// service version
	if err := writeUTF(buf, pkg.Service.Version); err != nil {
		return nil, err
	}
	// service method
	if err := writeUTF(buf, pkg.Service.Method); err != nil {
		return nil, err
	}
	// parameter types desc
	v := reflect.ValueOf(args[0])
	mv := v.MethodByName("JavaClassName")
	if mv.IsValid() {
		javaCls := mv.Call([]reflect.Value{})
		if len(javaCls) != 1 {
			return nil, errors.New("JavaStringName method should return exact 1 result")
		}
		javaClsStr, ok := javaCls[0].Interface().(string)
		if !ok {
			return nil, errors.New("JavaClassName method should return string")
		}
		if err := writeUTF(buf, getJavaArgType(javaClsStr)); err != nil {
			return nil, err
		}
	} else {
		// defensive code
		if err := writeUTF(buf, ""); err != nil {
			return nil, err
		}
	}
	// consumer args
	protoMsg := args[0].(proto.Message)
	if err := writeObject(buf, protoMsg); err != nil {
		return nil, err
	}
	// attachments
	atta := make(map[string]string)
	atta[PATH_KEY] = pkg.Service.Path
	atta[VERSION_KEY] = pkg.Service.Version
	if len(pkg.Service.Group) > 0 {
		atta[GROUP_KEY] = pkg.Service.Group
	}
	if len(pkg.Service.Interface) > 0 {
		atta[INTERFACE_KEY] = pkg.Service.Interface
	}
	if pkg.Service.Timeout != 0 {
		atta[TIMEOUT_KEY] = strconv.Itoa(int(pkg.Service.Timeout / time.Millisecond))
	}
	m := pb.Map{Attachments: atta}
	if err := writeObject(buf, &m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func marshalResponseProto(pkg DubboPackage) ([]byte, error) {
	response := EnsureResponsePayload(pkg.Body)
	buf := bytes.NewBuffer(make([]byte, 0))
	responseType := RESPONSE_VALUE
	hasAttachments := false
	if response.Attachments != nil {
		responseType = RESPONSE_VALUE_WITH_ATTACHMENTS
		hasAttachments = true
	} else {
		responseType = RESPONSE_VALUE
	}
	if response.Exception != nil {
		if hasAttachments {
			responseType = RESPONSE_WITH_EXCEPTION_WITH_ATTACHMENTS
		} else {
			responseType = RESPONSE_WITH_EXCEPTION
		}
	}
	// write response type
	if err := writeByte(buf, responseType); err != nil {
		return nil, err
	}
	if response.Exception != nil {
		// deal with exception
		throwable := pb.ThrowableProto{OriginalMessage: response.Exception.Error()}
		if err := writeObject(buf, &throwable); err != nil {
			return nil, err
		}
	} else {
		res, ok := response.RspObj.(proto.Message)
		if !ok {
			return nil, errors.New("proto buffer params should be marshaled in proto.Message")
		}
		// response body
		if err := writeObject(buf, res); err != nil {
			return nil, err
		}
	}

	if hasAttachments {
		attachments := pb.Map{Attachments: response.Attachments}
		if err := writeObject(buf, &attachments); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func init() {
	SetSerializer("protobuf", ProtoSerializer{})
}

func getJavaArgType(javaClsName string) string {
	return fmt.Sprintf("L%s;", strings.ReplaceAll(javaClsName, ".", "/"))
}

func writeUTF(writer io.Writer, value string) error {
	_, err := pbutil.WriteDelimited(writer, &pb.StringValue{Value: value})
	return err
}

func writeObject(writer io.Writer, value proto.Message) error {
	_, err := pbutil.WriteDelimited(writer, value)
	return err
}

func writeByte(writer io.Writer, v int32) error {
	i32v := &pb.Int32Value{Value: v}
	_, err := pbutil.WriteDelimited(writer, i32v)
	return err
}

func readUTF(reader io.Reader, value *string) error {
	sv := &pb.StringValue{}
	_, err := pbutil.ReadDelimited(reader, sv)
	if err != nil {
		return err
	}
	*value = sv.Value
	return nil
}

func readObject(reader io.Reader, value proto.Message) error {
	_, err := pbutil.ReadDelimited(reader, value)
	if err != nil {
		return err
	}
	return nil
}

// just as java protobuf serialize
func readByte(reader io.Reader, value *int32) error {
	i32v := &pb.Int32Value{}
	_, err := pbutil.ReadDelimited(reader, i32v)
	if err != nil {
		return err
	}
	*value = i32v.Value
	return nil
}

//
func readDelimitedLength(reader io.Reader, length *int) error {
	var headerBuf [binary.MaxVarintLen32]byte
	var bytesRead, varIntBytes int
	var messageLength uint64
	for varIntBytes == 0 { // i.e. no varint has been decoded yet.
		if bytesRead >= len(headerBuf) {
			return errors.New("invalid varint32 encountered")
		}
		// We have to read byte by byte here to avoid reading more bytes
		// than required. Each read byte is appended to what we have
		// read before.
		newBytesRead, err := reader.Read(headerBuf[bytesRead : bytesRead+1])
		if newBytesRead == 0 {
			if err != nil {
				return err
			}
			// A Reader should not return (0, nil), but if it does,
			// it should be treated as no-op (according to the
			// Reader contract). So let's go on...
			continue
		}
		bytesRead += newBytesRead
		// Now present everything read so far to the varint decoder and
		// see if a varint can be decoded already.
		messageLength, varIntBytes = proto.DecodeVarint(headerBuf[:bytesRead])
	}
	*length = int(messageLength)
	return nil
}

type JavaProto interface {
	JavaClassName() string
	proto.Message
}

type Register struct {
	sync.RWMutex
	registry map[string]reflect.Type
}

var (
	register = Register{
		registry: make(map[string]reflect.Type),
	}
)

func RegisterMessage(msg JavaProto) {
	register.Lock()
	defer register.Unlock()

	name := msg.JavaClassName()
	name = getJavaArgType(name)

	if e, ok := register.registry[name]; ok {
		panic(fmt.Sprintf("msg: %v has been registered. existed: %v", msg.JavaClassName(), e))
	}

	register.registry[name] = typeOfMessage(msg)
}

func getRegisterMessage(sig string) reflect.Value {
	register.Lock()
	defer register.Unlock()

	t, ok := register.registry[sig]
	if !ok {
		logger.Error(fmt.Sprintf("registry dose not have for svc: %v", sig))
		return NilValue
	}
	return reflect.New(t)
}

func typeOfMessage(o proto.Message) reflect.Type {
	v := reflect.ValueOf(o)
	switch v.Kind() {
	case reflect.Struct:
		return v.Type()
	case reflect.Ptr:
		return v.Elem().Type()
	}

	return reflect.TypeOf(o)
}
