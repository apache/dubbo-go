// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxtime encapsulates some golang.time functions
package gxtime

import (
	"time"
)

type Ticker struct {
	C  <-chan time.Time
	ID TimerID
	w  *TimerWheel
}

func NewTicker(d time.Duration) *Ticker {
	if d <= 0 {
		return nil
	}

	return defaultTimerWheel.NewTicker(d)
}

func TickFunc(d time.Duration, f func()) *Ticker {
	if d <= 0 {
		return nil
	}

	return defaultTimerWheel.TickFunc(d, f)
}

// 返回的channel无法被close
func Tick(d time.Duration) <-chan time.Time {
	if d <= 0 {
		return nil
	}

	return defaultTimerWheel.Tick(d)
}

func (t *Ticker) Stop() {
	(*Timer)(t).Stop()
}

func (t *Ticker) Reset(d time.Duration) {
	if d <= 0 {
		return
	}

	(*Timer)(t).Reset(d)
}
