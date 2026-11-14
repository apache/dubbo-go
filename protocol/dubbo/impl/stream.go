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
	"fmt"
	"reflect"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go-hessian2/java_exception"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

// StreamSerializer retains the Dubbo body layout and delegates its ordered
// objects to one exclusive encoder/decoder for the entire body.
type StreamSerializer struct{ Codec serialization.StreamCodec }

func (s StreamSerializer) Marshal(p DubboPackage) (out []byte, err error) {
	e, err := s.Codec.NewEncoder()
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := e.Close(); err == nil {
			err = closeErr
		}
	}()
	adapter := streamWriter{Encoder: e, hessian: s.Codec.Name() == "hessian2"}
	if p.IsHeartBeat() {
		if err = e.Encode(nil); err != nil {
			return nil, err
		}
		if adapter.hessian && p.Header.ResponseStatus != 0 {
			if err = e.Encode(nil); err != nil {
				return nil, err
			}
		}
		return e.Finish()
	}
	if p.IsRequest() && !p.IsHeartBeat() {
		err = encodeRequest(adapter, p)
	} else {
		err = encodeResponse(adapter, p, EnsureResponsePayload(p.Body))
		// Preserve the legacy Hessian response terminator. Other streams own their
		// termination and reference state through Finish.
		if err == nil && adapter.hessian && (!p.IsHeartBeat() || p.Header.ResponseStatus != 0) {
			err = e.Encode(nil)
		}
	}
	if err != nil {
		return nil, err
	}
	return e.Finish()
}
func (s StreamSerializer) Unmarshal(b []byte, p *DubboPackage) (err error) {
	defer func() {
		if invalid := recover(); invalid != nil {
			err = fmt.Errorf("invalid Dubbo stream body: %v", invalid)
		}
	}()
	d, err := s.Codec.NewDecoder(b)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := d.Close(); err == nil {
			err = closeErr
		}
	}()
	if p.IsHeartBeat() {
		var value any
		return d.Decode(&value)
	}
	if p.IsRequest() {
		if err := unmarshalStreamRequest(d, p); err != nil {
			return err
		}
		return s.decodeTypedArguments(b, p)
	}
	if p.IsResponseWithException() {
		var message string
		if err := d.Decode(&message); err != nil {
			return err
		}
		p.Body = &ResponsePayload{Exception: fmt.Errorf("java exception:%s", message)}
		return nil
	}
	return unmarshalStreamResponse(d, p, s.Codec.Name() == "hessian2")
}

// Protocol fields have fixed Go targets. In particular, response flags are
// int32 and attachment keys are strings; a generic codec need not recreate
// Hessian's dynamic int32 and map[any]any representation when decoding into any.
func unmarshalStreamRequest(d serialization.Decoder, p *DubboPackage) error {
	fields := make([]string, 5)
	for i := range fields {
		if err := d.Decode(&fields[i]); err != nil {
			return fmt.Errorf("decode request field %d: %w", i, err)
		}
	}
	descriptors := hessian.DescRegex.FindAllString(fields[4], -1)
	args := make([]any, len(descriptors))
	for i := range args {
		if err := d.Decode(&args[i]); err != nil {
			return fmt.Errorf("decode request argument %d: %w", i, err)
		}
	}
	var attachments map[string]any
	if err := d.Decode(&attachments); err != nil {
		return fmt.Errorf("decode request attachments: %w", err)
	}
	if attachments == nil {
		attachments = map[string]any{constant.InterfaceKey: fields[1]}
	}
	attachments[DUBBO_VERSION_KEY] = fields[0]
	p.Body = []any{fields[0], fields[1], fields[2], fields[3], fields[4], args, attachments}
	buildServerSidePackageBody(p)
	return nil
}

// Group and interface attachments follow the arguments on the wire, so the
// first pass resolves the exact exported service. If its signature requires
// concrete types, replay the original body into those targets. Converting the
// first pass's dynamic numbers would lose precision or silently truncate them.
func (s StreamSerializer) decodeTypedArguments(body []byte, p *DubboPackage) (err error) {
	request := p.Body.(map[string]any)
	service, _ := request["service"].(*common.Service)
	if service == nil {
		return nil
	} // Preserve legacy routing errors for unknown services.
	method := service.Method()[p.Service.Method]
	if method == nil {
		return nil
	}
	types := method.ArgsType()
	args := request["args"].([]any)
	// Legacy catch-all methods receive the entire dynamic argument list.
	if (len(types) == 1 || len(types) == 2 && method.ReplyType() == nil) && types[0] == reflect.TypeOf([]any{}) {
		return nil
	}
	if len(types) != len(args) {
		return fmt.Errorf("method %s expects %d arguments, received %d", p.Service.Method, len(types), len(args))
	}
	typed := false
	for i, arg := range args {
		if arg == nil || !reflect.TypeOf(arg).AssignableTo(types[i]) {
			typed = true
			break
		}
	}
	if !typed {
		return nil
	}
	decoder, err := s.Codec.NewDecoder(body)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := decoder.Close(); err == nil {
			err = closeErr
		}
	}()
	for i := 0; i < 5; i++ {
		var field string
		if err = decoder.Decode(&field); err != nil {
			return fmt.Errorf("decode request field %d: %w", i, err)
		}
	}
	for i, targetType := range types {
		target := reflect.New(targetType)
		if err = decoder.Decode(target.Interface()); err != nil {
			return fmt.Errorf("decode argument %d of %s as %s: %w", i, p.Service.Method, targetType, err)
		}
		args[i] = target.Elem().Interface()
	}
	return nil
}

func unmarshalStreamResponse(d serialization.Decoder, p *DubboPackage, isHessian bool) error {
	var flag int32
	if err := d.Decode(&flag); err != nil {
		return fmt.Errorf("decode response flag: %w", err)
	}
	response := EnsureResponsePayload(p.Body)
	p.Body = response
	withAttachments := false
	switch flag {
	case RESPONSE_WITH_EXCEPTION, RESPONSE_WITH_EXCEPTION_WITH_ATTACHMENTS:
		withAttachments = flag == RESPONSE_WITH_EXCEPTION_WITH_ATTACHMENTS
		if isHessian {
			var exception any
			if err := d.Decode(&exception); err != nil {
				return fmt.Errorf("decode response exception: %w", err)
			}
			if generic, ok := hessian.ToGenericException(exception); ok {
				response.Exception = generic
			} else if err, ok := exception.(error); ok {
				response.Exception = err
			} else {
				response.Exception = fmt.Errorf("got exception: %+v", exception)
			}
		} else {
			var exception string
			if err := d.Decode(&exception); err != nil {
				return fmt.Errorf("decode response exception: %w", err)
			}
			response.Exception = fmt.Errorf("%s", exception)
		}
	case RESPONSE_VALUE, RESPONSE_VALUE_WITH_ATTACHMENTS:
		withAttachments = flag == RESPONSE_VALUE_WITH_ATTACHMENTS
		target := response.RspObj
		var dynamic any
		if target == nil {
			target = &dynamic
		}
		if err := d.Decode(target); err != nil {
			return fmt.Errorf("decode response value: %w", err)
		}
		if response.RspObj == nil {
			response.RspObj = dynamic
		}
	case RESPONSE_NULL_VALUE, RESPONSE_NULL_VALUE_WITH_ATTACHMENTS:
		withAttachments = flag == RESPONSE_NULL_VALUE_WITH_ATTACHMENTS
	default:
		return fmt.Errorf("invalid response flag %d", flag)
	}
	if withAttachments {
		var attachments map[string]any
		if err := d.Decode(&attachments); err != nil {
			return fmt.Errorf("decode response attachments: %w", err)
		}
		response.Attachments = attachments
	}
	return nil
}

type streamWriter struct {
	serialization.Encoder
	hessian bool
}

func (w streamWriter) EncodeException(err error) error {
	if !w.hessian {
		return w.Encode(err.Error())
	}
	var value any
	switch ex := err.(type) {
	case *hessian.GenericException:
		value = java_exception.NewDubboGenericException(ex.ExceptionClass, ex.ExceptionMessage)
	case hessian.GenericException:
		value = java_exception.NewDubboGenericException(ex.ExceptionClass, ex.ExceptionMessage)
	case java_exception.Throwabler:
		value = ex
	default:
		value = java_exception.NewThrowable(err.Error())
	}
	return w.Encode(value)
}
