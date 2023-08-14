package config

import (
	"fmt"
	"regexp"
	"strings"
)

import (
	"github.com/go-playground/validator/v10"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// TranslateIds string "nacos,zk" => ["nacos","zk"]
func TranslateIds(registryIds []string) []string {
	ids := make([]string, 0)
	for _, id := range registryIds {

		ids = append(ids, strings.Split(id, ",")...)
	}
	return removeDuplicateElement(ids)
}

// removeDuplicateElement remove duplicate element
func removeDuplicateElement(items []string) []string {
	result := make([]string, 0, len(items))
	temp := map[string]struct{}{}
	for _, item := range items {
		if _, ok := temp[item]; !ok && item != "" {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func Verify(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		errs := err.(validator.ValidationErrors)
		var slice []string
		for _, msg := range errs {
			slice = append(slice, msg.Error())
		}
		return perrors.New(strings.Join(slice, ","))
	}
	return nil
}

func MergeValue(str1, str2, def string) string {
	if str1 == "" && str2 == "" {
		return def
	}
	s1 := strings.Split(str1, ",")
	s2 := strings.Split(str2, ",")
	str := "," + strings.Join(append(s1, s2...), ",")
	defKey := strings.Contains(str, ","+constant.DefaultKey)
	if !defKey {
		str = "," + constant.DefaultKey + str
	}
	str = strings.TrimPrefix(strings.Replace(str, ","+constant.DefaultKey, ","+def, -1), ",")
	return removeMinus(strings.Split(str, ","))
}

func removeMinus(strArr []string) string {
	if len(strArr) == 0 {
		return ""
	}
	var normalStr string
	var minusStrArr []string
	for _, v := range strArr {
		if strings.HasPrefix(v, "-") {
			minusStrArr = append(minusStrArr, v[1:])
		} else {
			normalStr += fmt.Sprintf(",%s", v)
		}
	}
	normalStr = strings.Trim(normalStr, ",")
	for _, v := range minusStrArr {
		normalStr = strings.Replace(normalStr, v, "", 1)
	}
	reg := regexp.MustCompile("[,]+")
	normalStr = reg.ReplaceAllString(strings.Trim(normalStr, ","), ",")
	return normalStr
}

func IsValid(addr string) bool {
	return addr != "" && addr != constant.NotAvailable
}
