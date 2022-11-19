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

package parser

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/oliveagle/jsonpath"
)

const (
	_pefixParam     = "param"
	_prefixParamArr = "param["
)

var (
	_arrayRegx, _ = regexp.Compile(`"^.+\\[[0-9]+\\]"`)
)

// ParseArgumentsByExpression follow https://goessner.net/articles/JsonPath/
//
//	{
//	    "store":{
//	        "book":[
//	            {
//	                "category":"reference",
//	                "author":"Nigel Rees",
//	                "title":"Sayings of the Century",
//	                "price":8.95
//	            },
//	            {
//	                "category":"fiction",
//	                "author":"Evelyn Waugh",
//	                "title":"Sword of Honor",
//	                "price":12.99
//	            },
//	            {
//	                "category":"fiction",
//	                "author":"Herman Melville",
//	                "title":"Moby Dick",
//	                "isbn":"0-553-21311-3",
//	                "price":8.99
//	            },
//	            {
//	                "category":"fiction",
//	                "author":"J. R. R. Tolkien",
//	                "title":"The Lord of the Rings",
//	                "isbn":"0-395-19395-8",
//	                "price":22.99
//	            }
//	        ],
//	        "bicycle":{
//	            "color":"red",
//	            "price":19.95
//	        }
//	    }
//	}
//
// examples
//   - case 1: param.$.store.book[*].author
func ParseArgumentsByExpression(key string, parameters []interface{}) interface{} {
	index, key := resolveIndex(key)
	if index == -1 || index >= len(parameters) {
		logger.Errorf("[Parser][Polaris] invalid expression for : %s", key)
		return nil
	}

	data, err := json.Marshal(parameters[index])
	if err != nil {
		logger.Errorf("[Parser][Polaris] marshal parameter %+v fail : %+v", parameters[index], err)
		return nil
	}
	var searchVal interface{}
	_ = json.Unmarshal(data, &searchVal)
	res, err := jsonpath.JsonPathLookup(searchVal, key)
	if err != nil {
		logger.Errorf("[Parser][Polaris] invalid do json path lookup by key : %s, err : %+v", key, err)
	}

	return res
}

func resolveIndex(key string) (int, string) {
	if strings.HasPrefix(key, _prefixParamArr) {
		// param[0].$.
		endIndex := strings.Index(key, "]")
		indexStr := key[len(_prefixParamArr):endIndex]
		index, err := strconv.ParseInt(indexStr, 10, 32)
		if err != nil {
			return -1, ""
		}
		startIndex := endIndex + 2
		if rune(key[endIndex+1]) != rune('.') {
			startIndex = endIndex + 1
		}
		return int(index), key[startIndex:]
	} else if strings.HasPrefix(key, _pefixParam) {
		key = strings.TrimPrefix(key, _pefixParam+".")
		return 0, strings.TrimPrefix(key, _pefixParam+".")
	}

	return -1, ""
}
