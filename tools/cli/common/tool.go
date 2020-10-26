package common

import (
	"fmt"
	"log"
	"reflect"
)

// PrintInterface 层级打印struct
func PrintInterface(v interface{}) {
	val := reflect.ValueOf(v).Elem()
	typ := reflect.TypeOf(v)
	log.Printf("%+v\n", v)
	nums := val.NumField()
	for i := 0; i < nums; i++ {
		if typ.Elem().Field(i).Type.Kind() == reflect.Ptr {
			log.Printf("%s: ", typ.Elem().Field(i).Name)
			PrintInterface(val.Field(i).Interface())
		}
	}
	fmt.Println("")
}
