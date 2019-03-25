// +build !amd64

// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.
//
// refers to github.com/jonhoo/drwmutex
package gxsync

// cpu returns a unique identifier for the core the current goroutine is
// executing on. This function is platform dependent, and is implemented in
// cpu_*.s.
func cpu() uint64 {
	// this reverts the behaviour to that of a regular DRWMutex
	return 0
}
