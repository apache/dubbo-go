// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxtime encapsulates some golang.time functions
package gxtime

import (
	"time"
)

/////////////////////////////////////////
// count watch
/////////////////////////////////////////

type CountWatch struct {
	start time.Time
}

func (w *CountWatch) Start() {
	var t time.Time
	if t.Equal(w.start) {
		w.start = time.Now()
	}
}

func (w *CountWatch) Reset() {
	w.start = time.Now()
}

func (w *CountWatch) Count() int64 {
	return time.Since(w.start).Nanoseconds()
}
