// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxtime encapsulates some golang.time functions
// ref: https://github.com/AlexStocks/go-practice/blob/master/time/siddontang_time_wheel.go
package gxtime

import (
	//"fmt"
	"sync"
	"time"
)

type Wheel struct {
	sync.RWMutex
	span   time.Duration
	period time.Duration
	ticker *time.Ticker
	index  int
	ring   []chan struct{}
	once   sync.Once
	now    time.Time
}

func NewWheel(span time.Duration, buckets int) *Wheel {
	var (
		w *Wheel
	)

	if span == 0 {
		panic("@span == 0")
	}
	if buckets == 0 {
		panic("@bucket == 0")
	}

	w = &Wheel{
		span:   span,
		period: span * (time.Duration(buckets)),
		ticker: time.NewTicker(span),
		index:  0,
		ring:   make([](chan struct{}), buckets),
		now:    time.Now(),
	}

	go func() {
		var notify chan struct{}
		// var cw CountWatch
		// cw.Start()
		for t := range w.ticker.C {
			w.Lock()
			w.now = t

			// fmt.Println("index:", w.index, ", value:", w.bitmap.Get(w.index))
			notify = w.ring[w.index]
			w.ring[w.index] = nil
			w.index = (w.index + 1) % len(w.ring)

			w.Unlock()

			if notify != nil {
				close(notify)
			}
		}
		// fmt.Println("timer costs:", cw.Count()/1e9, "s")
	}()

	return w
}

func (w *Wheel) Stop() {
	w.once.Do(func() { w.ticker.Stop() })
}

func (w *Wheel) After(timeout time.Duration) <-chan struct{} {
	if timeout >= w.period {
		panic("@timeout over ring's life period")
	}

	var pos = int(timeout / w.span)
	if 0 < pos {
		pos--
	}

	w.Lock()
	pos = (w.index + pos) % len(w.ring)
	if w.ring[pos] == nil {
		w.ring[pos] = make(chan struct{})
	}
	// fmt.Println("pos:", pos)
	c := w.ring[pos]
	w.Unlock()

	return c
}

func (w *Wheel) Now() time.Time {
	w.RLock()
	now := w.now
	w.RUnlock()

	return now
}
