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

package yaml

import (
	"path/filepath"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalYMLConfig(t *testing.T) {
	conPath, err := filepath.Abs("./testdata/config.yml")
	assert.NoError(t, err)
	c := &Config{}
	_, err = UnmarshalYMLConfig(conPath, c)
	assert.NoError(t, err)
	assert.Equal(t, "strTest", c.StrTest)
	assert.Equal(t, 11, c.IntTest)
	assert.Equal(t, false, c.BooleanTest)
	assert.Equal(t, "childStrTest", c.ChildConfig.StrTest)
}

func TestUnmarshalYMLConfigError(t *testing.T) {
	c := &Config{}
	_, err := UnmarshalYMLConfig("./testdata/config", c)
	assert.Error(t, err)
	_, err = UnmarshalYMLConfig("", c)
	assert.Error(t, err)
}

func TestUnmarshalYML(t *testing.T) {
	c := &Config{}
	b, err := LoadYMLConfig("./testdata/config.yml")
	assert.NoError(t, err)
	err = UnmarshalYML(b, c)
	assert.NoError(t, err)
	assert.Equal(t, "strTest", c.StrTest)
	assert.Equal(t, 11, c.IntTest)
	assert.Equal(t, false, c.BooleanTest)
	assert.Equal(t, "childStrTest", c.ChildConfig.StrTest)
}

type Config struct {
	StrTest     string      `yaml:"strTest" default:"default" json:"strTest,omitempty" property:"strTest"`
	IntTest     int         `default:"109"  yaml:"intTest" json:"intTest,omitempty" property:"intTest"`
	BooleanTest bool        `yaml:"booleanTest" default:"true" json:"booleanTest,omitempty"`
	ChildConfig ChildConfig `yaml:"child" json:"child,omitempty"`
}

type ChildConfig struct {
	StrTest string `default:"strTest" default:"default" yaml:"strTest"  json:"strTest,omitempty"`
}
