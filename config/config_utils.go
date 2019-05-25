package config

import (
	"regexp"
	"strings"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
)

func mergeValue(str1, str2, def string) string {
	if str1 == "" && str2 == "" {
		return def
	}
	str := "," + strings.Trim(str1, ",")
	if str1 == "" {
		str = "," + strings.Trim(str2, ",")
	} else if str2 != "" {
		str = str + "," + strings.Trim(str2, ",")
	}
	defKey := strings.Contains(str, ","+constant.DEFAULT_KEY)
	if !defKey {
		str = "," + constant.DEFAULT_KEY + str
	}
	str = strings.TrimPrefix(strings.Replace(str, ","+constant.DEFAULT_KEY, ","+def, -1), ",")

	strArr := strings.Split(str, ",")
	strMap := make(map[string][]int)
	for k, v := range strArr {
		add := true
		if strings.HasPrefix(v, "-") {
			v = v[1:]
			add = false
		}
		if _, ok := strMap[v]; !ok {
			if add {
				strMap[v] = []int{1, k}
			}
		} else {
			if add {
				strMap[v][0] += 1
				strMap[v] = append(strMap[v], k)
			} else {
				strMap[v][0] -= 1
				strMap[v] = strMap[v][:len(strMap[v])-1]
			}
		}
	}
	strArr = make([]string, len(strArr))
	for key, value := range strMap {
		if value[0] == 0 {
			continue
		}
		for i := 1; i < len(value); i++ {
			strArr[value[i]] = key
		}
	}
	reg := regexp.MustCompile("[,]+")
	str = reg.ReplaceAllString(strings.Join(strArr, ","), ",")
	return strings.Trim(str, ",")
}
