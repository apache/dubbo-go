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
	 "github.com/stretchr/testify/assert"
 )

 import (
	"github.com/dubbogo/gost/container/set"
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