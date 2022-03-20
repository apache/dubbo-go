package xds

import (
	"encoding/json"
)

type ADSZResponse struct {
	Clients []ADSZClient `json:"clients"`
}

type ADSZClient struct {
	Metadata map[string]interface{} `json:"metadata"`
}

func (a *ADSZResponse) GetMap() map[string]string {
	result := make(map[string]string)
	for _, c := range a.Clients {
		resultMap := make(map[string]string)
		json.Unmarshal([]byte(c.Metadata["LABELS"].(map[string]interface{})["DUBBO_GO"].(string)), &resultMap)
		for k, v := range resultMap {
			result[k] = v
		}
	}
	return result
}
