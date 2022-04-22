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

package mapping

import (
	"encoding/json"
)

// ADSZResponse is body got from istiod:8080/debug/adsz
type ADSZResponse struct {
	Clients []ADSZClient `json:"clients"`
}

type ADSZClient struct {
	Metadata map[string]interface{} `json:"metadata"`
}

func (a *ADSZResponse) GetMap() map[string]string {
	result := make(map[string]string)
	for _, c := range a.Clients {
		if c.Metadata["LABELS"] == nil {
			continue
		}
		labelsMap, ok := c.Metadata["LABELS"].(map[string]interface{})
		if !ok {
			continue
		}
		dubbogoMetadata := labelsMap["DUBBO_GO"]
		if dubbogoMetadata == nil {
			continue
		}
		dubbogoMetadataStr, ok := dubbogoMetadata.(string)
		if !ok {
			continue
		}
		resultMap := make(map[string]string)
		_ = json.Unmarshal([]byte(dubbogoMetadataStr), &resultMap)
		for k, v := range resultMap {
			result[k] = v
		}
	}
	return result
}
