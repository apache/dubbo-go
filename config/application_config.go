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

package config

import (
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
)

func NewApplicationConfigBuilder() *ApplicationConfigBuilder {
	return &ApplicationConfigBuilder{application: &commonCfg.ApplicationConfig{}}
}

type ApplicationConfigBuilder struct {
	application *commonCfg.ApplicationConfig
}

func (acb *ApplicationConfigBuilder) SetOrganization(organization string) *ApplicationConfigBuilder {
	acb.application.Organization = organization
	return acb
}

func (acb *ApplicationConfigBuilder) SetName(name string) *ApplicationConfigBuilder {
	acb.application.Name = name
	return acb
}

func (acb *ApplicationConfigBuilder) SetModule(module string) *ApplicationConfigBuilder {
	acb.application.Module = module
	return acb
}

func (acb *ApplicationConfigBuilder) SetVersion(version string) *ApplicationConfigBuilder {
	acb.application.Version = version
	return acb
}

func (acb *ApplicationConfigBuilder) SetOwner(owner string) *ApplicationConfigBuilder {
	acb.application.Owner = owner
	return acb
}

func (acb *ApplicationConfigBuilder) SetEnvironment(environment string) *ApplicationConfigBuilder {
	acb.application.Environment = environment
	return acb
}

func (acb *ApplicationConfigBuilder) SetMetadataType(metadataType string) *ApplicationConfigBuilder {
	acb.application.MetadataType = metadataType
	return acb
}

func (acb *ApplicationConfigBuilder) Build() *commonCfg.ApplicationConfig {
	return acb.application
}
