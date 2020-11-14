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

package kubernetes

import (
	"strconv"
	"sync"
	"testing"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

type mockFacade struct {
	*common.URL
	client  *Client
	cltLock sync.Mutex
	done    chan struct{}
}

func (r *mockFacade) Client() *Client {
	return r.client
}

func (r *mockFacade) SetClient(client *Client) {
	r.client = client
}

func (r *mockFacade) GetUrl() *common.URL {
	return r.URL
}

func (r *mockFacade) Destroy() {
	// TODO implementation me
}

func (r *mockFacade) RestartCallBack() bool {
	return true
}

func (r *mockFacade) IsAvailable() bool {
	return true
}
func Test_Facade(t *testing.T) {

	regUrl, err := common.NewURL("registry://127.0.0.1:443",
		common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.CONSUMER)))
	if err != nil {
		t.Fatal(err)
	}

	mockClient := getTestClient(t)
	m := &mockFacade{
		URL:    regUrl,
		client: mockClient,
	}

	if err := ValidateClient(m); err == nil {
		t.Fatal("out of cluster should err")
	}
	mockClient.Close()
}
