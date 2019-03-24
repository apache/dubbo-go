// Copyright 2015 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

package gxxorlist

import (
	"fmt"
)

func OutputElem(e *XorElement) {
	if e != nil {
		// fmt.Printf("addr:%p, value:%v", e, e)
		fmt.Printf("value:%v", e.Value)
	}
}

// Iterate through list and print its contents.
func OutputList(l *XorList) {
	idx := 0
	for e, p := l.Front(); e != nil; p, e = e, e.Next(p) {
		fmt.Printf("idx:%v, ", idx)
		OutputElem(e)
		fmt.Printf("\n")
		idx++
	}
}

// Iterate through list and print its contents in reverse.
func OutputListR(l *XorList) {
	idx := 0
	for e, n := l.Back(); e != nil; e, n = e.Next(n), e {
		fmt.Printf("idx:%v, ", idx)
		OutputElem(e)
		fmt.Printf("\n")
		idx++
	}
}
