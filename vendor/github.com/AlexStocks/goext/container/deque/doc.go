// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// The gxdeque package implements an efficient double-ended queue data
// structure called Deque.
//
// Usage:
//
//    d := gxdeque.New()
//    d.PushFront("foo")
//    d.PushBack("bar")
//    d.PushBack("123")
//    l := d.Len()          // l == 3
//    v, ok := d.PopFront() // v.(string) == "foo", ok == true
//    v, ok = d.PopFront()  // v.(string) == "bar", ok == true
//    v, ok = d.PopBack()   // v.(string) == "123", ok == true
//    v, ok = d.PopBack()   // v == nil, ok == false
//    v, ok = d.PopFront()  // v == nil, ok == false
//    l = d.Len()           // l == 0
//
// A discussion of the internals can be found at the top of deque.go.
// ref: https://github.com/juju/utils/blob/master/deque/doc.go
package gxdeque
