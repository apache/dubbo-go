// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

package gxstrings

import (
	"reflect"
)

func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}

	if reflect.ValueOf(i).IsNil() {
		return true
	}

	return false
}
