// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// packaeg gxos encapsulates os related functions.
package gxos

import (
	"os"
	"reflect"
	"strings"
)

func CreateDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return err
		}
	}

	return nil
}

// get a struct object(or its ptr) @v's package path
func GetPkgPath(v interface{}) string {
	var (
		path  string
		value reflect.Value
	)

	value = reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.Struct:
		path = value.Type().PkgPath()
	case reflect.Ptr:
		path = value.Elem().Type().PkgPath()
	default:
		panic("err type")
	}
	moudle := strings.Split(path, "/")

	if len(moudle) > 0 {
		if moudle[len(moudle)-1] == "common" {
			panic(path)
		}
		return moudle[len(moudle)-1]
	}

	return path
}
