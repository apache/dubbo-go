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

package tag

import (
	"testing"
)

import (
	"github.com/stretchr/testify/suite"
)

type RuleTestSuite struct {
	suite.Suite
	rule *RouterRule
}

func (suite *RuleTestSuite) SetupTest() {
	var err error
	yml := `
scope: application
force: true
runtime: false
enabled: true
priority: 1
key: demo-provider
tags:
  - name: tag1
    addresses: [ip1, ip2]
  - name: tag2
    addresses: [ip3, ip4]
`
	suite.rule, err = getRule(yml)
	suite.Nil(err)
}

func (suite *RuleTestSuite) TestGetRule() {
	var err error
	suite.Equal(true, suite.rule.Force)
	suite.Equal(false, suite.rule.Runtime)
	suite.Equal("application", suite.rule.Scope)
	suite.Equal(1, suite.rule.Priority)
	suite.Equal("demo-provider", suite.rule.Key)
	suite.Nil(err)
}

func (suite *RuleTestSuite) TestGetTagNames() {
	suite.Equal([]string{"tag1", "tag2"}, suite.rule.getTagNames())
}

func (suite *RuleTestSuite) TestGetAddresses() {
	suite.Equal([]string{"ip1", "ip2", "ip3", "ip4"}, suite.rule.getAddresses())
}

func (suite *RuleTestSuite) TestHasTag() {
	suite.Equal(true, suite.rule.hasTag("tag1"))
	suite.Equal(false, suite.rule.hasTag("tag404"))
}

func TestRuleTestSuite(t *testing.T) {
	suite.Run(t, new(RuleTestSuite))
}
