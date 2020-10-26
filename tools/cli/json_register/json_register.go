package json_register

import (
	"fmt"
	"log"
	"reflect"

	"github.com/edison/go-telnet/common"

	hessian "github.com/LaurenceLiZhixin/dubbo-go-hessian2"
	jparser "github.com/LaurenceLiZhixin/json-interface-parser"
)

func RegisterStructFromFile(path string) interface{} {
	if path == "" {
		return nil
	}
	pkg, err := jparser.JsonFile2Interface(path)
	log.Printf("Created pkg: \n")
	common.PrintInterface(pkg)
	if err != nil {
		fmt.Println("error: json file parse failed :", err)
		return nil
	}
	hessian.RegisterPOJOMapping(getJavaClassName(pkg), pkg)
	return pkg
}

func getJavaClassName(pkg interface{}) string {
	val := reflect.ValueOf(pkg).Elem()
	typ := reflect.TypeOf(pkg).Elem()
	nums := val.NumField()
	for i := 0; i < nums; i++ {
		if typ.Field(i).Name == "JavaClassName" {
			return val.Field(i).String()
		}
	}
	fmt.Println("error: JavaClassName not found")
	return ""
}
