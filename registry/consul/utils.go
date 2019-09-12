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

package consul

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strconv"
)

import (
	consul "github.com/hashicorp/consul/api"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/utils"
)

func buildId(url common.URL) string {
	t := md5.Sum([]byte(url.String()))
	return hex.EncodeToString(t[:])
}

func buildService(url common.URL) (*consul.AgentServiceRegistration, error) {
	var err error

	// id
	id := buildId(url)

	// address
	if url.Ip == "" {
		url.Ip, _ = utils.GetLocalIP()
	}

	// port
	port, err := strconv.Atoi(url.Port)
	if err != nil {
		return nil, err
	}

	// tcp
	tcp := fmt.Sprintf("%s:%d", url.Ip, port)

	// tags
	tags := make([]string, 0, 8)

	url.RangeParams(func(key, value string) bool {
		tags = append(tags, key+"="+value)
		return true
	})

	tags = append(tags, "dubbo")

	// meta
	meta := make(map[string]string, 8)
	meta["url"] = url.String()

	// check
	check := &consul.AgentServiceCheck{
		TCP:                            tcp,
		Interval:                       url.GetParam("consul-check-interval", "10s"),
		Timeout:                        url.GetParam("consul-check-timeout", "1s"),
		DeregisterCriticalServiceAfter: url.GetParam("consul-deregister-critical-service-after", "20s"),
	}

	service := &consul.AgentServiceRegistration{
		Name:    url.Service(),
		ID:      id,
		Address: url.Ip,
		Port:    port,
		Tags:    tags,
		Meta:    meta,
		Check:   check,
	}

	return service, nil
}

func retrieveURL(service *consul.ServiceEntry) (common.URL, error) {
	url, ok := service.Service.Meta["url"]
	if !ok {
		return common.URL{}, perrors.New("retrieve url fails with no url key in service meta")
	}
	url1, err := common.NewURL(context.Background(), url)
	if err != nil {
		return common.URL{}, perrors.WithStack(err)
	}
	return url1, nil
}

func in(url common.URL, urls []common.URL) bool {
	for _, url1 := range urls {
		if url.URLEqual(url1) {
			return true
		}
	}
	return false
}
