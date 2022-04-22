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

package server

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config"
)

func TestGoRestfulServerDeploySameUrl(t *testing.T) {
	grs := NewGoRestfulServer()
	url, err := common.NewURL("http://127.0.0.1:43121")
	assert.NoError(t, err)
	grs.Start(url)
	rmc := &config.RestMethodConfig{
		Produces:   "*/*",
		Consumes:   "*/*",
		MethodType: "POST",
		Path:       "/test",
	}
	f := func(request RestServerRequest, response RestServerResponse) {}
	grs.Deploy(rmc, f)
	rmc1 := &config.RestMethodConfig{
		Produces:   "*/*",
		Consumes:   "*/*",
		MethodType: "GET",
		Path:       "/test",
	}
	grs.Deploy(rmc1, f)
	grs.UnDeploy(rmc)
	grs.UnDeploy(rmc1)
	grs.Destroy()
}
