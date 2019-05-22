// Copyright 2016-2019 hxmhlt
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extension

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/registry"
)

var (
	registrys = make(map[string]func(config *common.URL) (registry.Registry, error))
)

func SetRegistry(name string, v func(config *common.URL) (registry.Registry, error)) {
	registrys[name] = v
}

func GetRegistry(name string, config *common.URL) (registry.Registry, error) {
	if registrys[name] == nil {
		panic("registry for " + name + " is not existing, make sure you have import the package.")
	}
	return registrys[name](config)

}
