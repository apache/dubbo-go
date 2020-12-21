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

package condition

import (
	"testing"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/dubbogo/gost/container/set"
	"github.com/stretchr/testify/assert"
)

func TestParseRule(t *testing.T) {
	testString := ``
	matchPair, err := parseRule(testString)
	assert.Nil(t, err)
	assert.EqualValues(t, matchPair, make(map[string]MatchPair, 16))

	testString = `method!=sayHello&application=sayGoodBye`
	matchPair, err = parseRule(testString)
	assert.Nil(t, err)
	assert.EqualValues(t, matchPair["method"].Mismatches, gxset.NewSet("sayHello"))
	assert.EqualValues(t, matchPair["application"].Matches, gxset.NewSet("sayGoodBye"))

	testString = `noRule`
	matchPair, err = parseRule(testString)
	assert.Nil(t, err)
	assert.EqualValues(t, matchPair["noRule"].Mismatches, gxset.NewSet())
	assert.EqualValues(t, matchPair["noRule"].Matches, gxset.NewSet())

	testString = `method!=sayHello,sayGoodBye`
	matchPair, err = parseRule(testString)
	assert.Nil(t, err)
	assert.EqualValues(t, matchPair["method"].Mismatches, gxset.NewSet(`sayHello`, `sayGoodBye`))

	testString = `method!=sayHello,sayGoodDay=sayGoodBye`
	matchPair, err = parseRule(testString)
	assert.Nil(t, err)
	assert.EqualValues(t, matchPair["method"].Mismatches, gxset.NewSet(`sayHello`, `sayGoodDay`))
	assert.EqualValues(t, matchPair["method"].Matches, gxset.NewSet(`sayGoodBye`))
}

func TestNewConditionRouter(t *testing.T) {
	url, _ := common.NewURL(`condition://0.0.0.0:?application=mock-app&category=routers&force=true&priority=1&router=condition&rule=YSAmIGMgPT4gYiAmIGQ%3D`)
	router, err := NewConditionRouter(url)
	assert.Nil(t, err)
	assert.Equal(t, true, router.Enabled())
	assert.Equal(t, true, router.Force)
	assert.Equal(t, int64(1), router.Priority())
	whenRule, _ := parseRule("a & c")
	thenRule, _ := parseRule("b & d")
	assert.EqualValues(t, router.WhenCondition, whenRule)
	assert.EqualValues(t, router.ThenCondition, thenRule)

	router, err = NewConditionRouter(nil)
	assert.Error(t, err)

	url, _ = common.NewURL(`condition://0.0.0.0:?application=mock-app&category=routers&force=true&priority=1&router=condition&rule=YSAmT4gYiAmIGQ%3D`)
	router, err = NewConditionRouter(url)
	assert.Error(t, err)

	url, _ = common.NewURL(`condition://0.0.0.0:?application=mock-app&category=routers&force=true&router=condition&rule=YSAmIGMgPT4gYiAmIGQ%3D`)
	router, err = NewConditionRouter(url)
	assert.Nil(t, err)
	assert.Equal(t, int64(150), router.Priority())

	url, _ = common.NewURL(`condition://0.0.0.0:?category=routers&force=true&interface=mock-service&router=condition&rule=YSAmIGMgPT4gYiAmIGQ%3D`)
	router, err = NewConditionRouter(url)
	assert.Nil(t, err)
	assert.Equal(t, int64(140), router.Priority())
}
