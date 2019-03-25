// +build !linux

// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.
//
// refers to github.com/jonhoo/drwmutex
package gxsync

func map_cpus() (cpus map[uint64]int) {
	cpus = make(map[uint64]int)
	return
}
