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

package client_impl

import (
	"context"
	"net"
	"net/http"
	"path"
	"time"
)

import (
	"github.com/go-resty/resty/v2"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/rest/client"
)

//func init() {
//	extension.SetRestClient(constant.DefaultRestClient, NewRestyClient)
//}
// RestyClient a rest client implement by Resty
type RestyClient struct {
	client *resty.Client
}

// NewRestyClient a constructor of RestyClient
func NewRestyClient(restOption *client.RestOptions) client.RestClient {
	client := resty.New()
	client.SetTransport(
		&http.Transport{
			DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(network, addr, restOption.ConnectTimeout)
				if err != nil {
					return nil, err
				}
				err = c.SetDeadline(time.Now().Add(restOption.RequestTimeout))
				if err != nil {
					return nil, err
				}
				return c, nil
			},
		})
	return &RestyClient{
		client: client,
	}
}

// Do send request by RestyClient
func (rc *RestyClient) Do(restRequest *client.RestClientRequest, res interface{}) error {
	req := rc.client.R()
	req.Header = restRequest.Header
	req.SetPathParams(restRequest.PathParams).
		SetQueryParams(restRequest.QueryParams).
		SetBody(restRequest.Body)
	if res != nil {
		req.SetResult(res)
	}
	resp, err := req.Execute(restRequest.Method, "http://"+path.Join(restRequest.Location, restRequest.Path))
	if err != nil {
		return perrors.WithStack(err)
	}
	if resp.IsError() {
		return perrors.New(resp.String())
	}
	return nil
}
