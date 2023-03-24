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

package generalizer

import (
	"encoding/json"
	"reflect"
	"sync"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo/hessian2"
)

var (
	jsonGeneralizer     Generalizer
	jsonGeneralizerOnce sync.Once
)

func GetGsonGeneralizer() Generalizer {
	jsonGeneralizerOnce.Do(func() {
		jsonGeneralizer = &GsonGeneralizer{}
	})
	return jsonGeneralizer
}

type GsonGeneralizer struct{}

func (GsonGeneralizer) Generalize(obj interface{}) (interface{}, error) {
	newObj, ok := obj.(hessian.POJO)
	if !ok {
		return nil, perrors.Errorf("unexpected type of obj(=%T), wanted is hessian pojo", obj)
	}

	jsonbytes, err := json.Marshal(newObj)
	if err != nil {
		return nil, err
	}

	return string(jsonbytes), nil
}

func (GsonGeneralizer) Realize(obj interface{}, typ reflect.Type) (interface{}, error) {
	jsonbytes, ok := obj.(string)
	if !ok {
		return nil, perrors.Errorf("unexpected type of obj(=%T), wanted is string", obj)
	}

	// create the target object
	ret, ok := reflect.New(typ).Interface().(hessian.POJO)
	if !ok {
		return nil, perrors.Errorf("the type of obj(=%s) should be hessian pojo", typ)
	}

	err := json.Unmarshal([]byte(jsonbytes), ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (GsonGeneralizer) GetType(obj interface{}) (typ string, err error) {
	typ, err = hessian2.GetJavaName(obj)
	// no error or error is not NilError
	if err == nil || err != hessian2.NilError {
		return
	}

	typ = "java.lang.Object"
	if err == hessian2.NilError {
		logger.Debugf("the type of nil object couldn't be inferred, use the default value(\"%s\")", typ)
		return
	}

	logger.Debugf("the type of object(=%T) couldn't be recognized as a POJO, use the default value(\"%s\")", obj, typ)
	return
}
