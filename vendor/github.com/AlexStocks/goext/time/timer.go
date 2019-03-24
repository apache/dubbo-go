// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxtime encapsulates some golang.time functions
package gxtime

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/AlexStocks/goext/container/xorlist"
	"github.com/AlexStocks/goext/log"
)

var (
	ErrTimeChannelFull   = fmt.Errorf("timer channel full")
	ErrTimeChannelClosed = fmt.Errorf("timer channel closed")
)

// to init a default timer wheel
func Init() {
	defaultTimerWheelOnce.Do(func() {
		defaultTimerWheel = NewTimerWheel()
	})
}

func Now() time.Time {
	return defaultTimerWheel.Now()
}

////////////////////////////////////////////////
// timer node
////////////////////////////////////////////////

var (
	defaultTimerWheelOnce sync.Once
	defaultTimerWheel     *TimerWheel
	nextID                TimerID
	curGxTime             = time.Now().UnixNano() // current goext time in nanoseconds
)

const (
	maxMS     = 1000
	maxSecond = 60
	maxMinute = 60
	maxHour   = 24
	maxDay    = 31
	// ticker interval不能设置到这种精度，
	// 实际运行时ticker的时间间隔会在1.001ms上下浮动,
	// 当ticker interval小于1ms的时候，会导致TimerWheel.hand
	// 和timeWheel.inc不增长，造成时间错乱：例如本来
	// 1.5s运行的函数在持续2.1s之后才被执行
	// minDiff = 1.001 * MS
	minDiff       = 10e6
	maxTimerLevel = 5
)

func msNum(expire int64) int64     { return expire / int64(time.Millisecond) }
func secondNum(expire int64) int64 { return expire / int64(time.Minute) }
func minuteNum(expire int64) int64 { return expire / int64(time.Minute) }
func hourNum(expire int64) int64   { return expire / int64(time.Hour) }
func dayNum(expire int64) int64    { return expire / (maxHour * int64(time.Hour)) }

// if the return error is not nil, the related timer will be closed.
type TimerFunc func(ID TimerID, expire time.Time, arg interface{}) error

type TimerID = uint64

type timerNode struct {
	ID       TimerID
	trig     int64
	typ      TimerType
	period   int64
	timerRun TimerFunc
	arg      interface{}
}

func newTimerNode(f TimerFunc, typ TimerType, period int64, arg interface{}) timerNode {
	return timerNode{
		ID:       atomic.AddUint64(&nextID, 1),
		trig:     atomic.LoadInt64(&curGxTime) + period,
		typ:      typ,
		period:   period,
		timerRun: f,
		arg:      arg,
	}
}

func compareTimerNode(first, second timerNode) int {
	var ret int

	if first.trig < second.trig {
		ret = -1
	} else if first.trig > second.trig {
		ret = 1
	} else {
		ret = 0
	}

	return ret
}

type timerAction = int64

const (
	ADD_TIMER   timerAction = 1
	DEL_TIMER   timerAction = 2
	RESET_TIMER timerAction = 3
)

type timerNodeAction struct {
	node   timerNode
	action timerAction
}

////////////////////////////////////////////////
// timer wheel
////////////////////////////////////////////////

const (
	timerNodeQueueSize = 128
)

var (
	limit   = [maxTimerLevel + 1]int64{maxMS, maxSecond, maxMinute, maxHour, maxDay}
	msLimit = [maxTimerLevel + 1]int64{
		int64(time.Millisecond),
		int64(time.Second),
		int64(time.Minute),
		int64(time.Hour),
		int64(maxHour * time.Hour),
	}
)

// timer based on multiple wheels
type TimerWheel struct {
	start  int64                             // start clock
	clock  int64                             // current time in nanosecond
	number int64                             // timer node number
	hand   [maxTimerLevel]int64              // clock
	slot   [maxTimerLevel]*gxxorlist.XorList // timer list

	timerQ chan timerNodeAction

	once   sync.Once // for close ticker
	ticker *time.Ticker
	wg     sync.WaitGroup
}

func NewTimerWheel() *TimerWheel {
	w := &TimerWheel{
		clock:  atomic.LoadInt64(&curGxTime),
		ticker: time.NewTicker(time.Duration(minDiff)), // 这个精度如果太低，会影响curGxTime，进而影响timerNode的trig的值
		timerQ: make(chan timerNodeAction, timerNodeQueueSize),
	}
	w.start = w.clock

	for i := 0; i < maxTimerLevel; i++ {
		w.slot[i] = gxxorlist.New()
	}

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		var (
			t          time.Time
			cFlag      bool
			nodeAction timerNodeAction
			qFlag      bool
		)

	LOOP:
		for {
			select {
			case t, cFlag = <-w.ticker.C:
				atomic.StoreInt64(&curGxTime, t.UnixNano())
				if cFlag && 0 != atomic.LoadInt64(&w.number) {
					ret := w.timerUpdate(t)
					if ret == 0 {
						w.run()
					}

					continue
				}

				break LOOP

			case nodeAction, qFlag = <-w.timerQ:
				// 此处只用一个channel，保证对同一个timer操作的顺序性
				if qFlag {
					switch {
					case nodeAction.action == ADD_TIMER:
						atomic.AddInt64(&w.number, 1)
						w.insertTimerNode(nodeAction.node)
					case nodeAction.action == DEL_TIMER:
						atomic.AddInt64(&w.number, -1)
						w.deleteTimerNode(nodeAction.node)
					case nodeAction.action == RESET_TIMER:
						// gxlog.CInfo("node action:%#v", nodeAction)
						w.resetTimerNode(nodeAction.node)
					default:
						atomic.AddInt64(&w.number, 1)
						w.insertTimerNode(nodeAction.node)
					}
					continue
				}

				break LOOP
			}
		}
	}()

	return w
}

func (w *TimerWheel) output() {
	for idx := range w.slot {
		gxlog.CDebug("print slot %d\n", idx)
		w.slot[idx].Output()
	}
}

func (w *TimerWheel) TimerNumber() int {
	return int(atomic.LoadInt64(&w.number))
}

func (w *TimerWheel) Now() time.Time {
	return UnixNano2Time(atomic.LoadInt64(&curGxTime))
}

func (w *TimerWheel) run() {
	var (
		clock int64
		err   error
		node  timerNode
		slot  *gxxorlist.XorList
		array []timerNode
	)

	slot = w.slot[0]
	clock = atomic.LoadInt64(&w.clock)
	for e, p := slot.Front(); e != nil; p, e = e, e.Next(p) {
		node = e.Value.(timerNode)
		if clock < node.trig {
			break
		}

		err = node.timerRun(node.ID, UnixNano2Time(clock), node.arg)
		if err == nil && node.typ == eTimerLoop {
			array = append(array, node)
			// w.insertTimerNode(node)
		} else {
			atomic.AddInt64(&w.number, -1)
		}

		temp := e
		e, p = p, p.Prev(e)
		slot.Remove(temp)
	}
	for idx := range array[:] {
		array[idx].trig += array[idx].period
		w.insertTimerNode(array[idx])
	}
}

func (w *TimerWheel) insertSlot(idx int, node timerNode) {
	var (
		pos  *gxxorlist.XorElement
		slot *gxxorlist.XorList
	)

	slot = w.slot[idx]
	for e, p := slot.Front(); e != nil; p, e = e, e.Next(p) {
		if compareTimerNode(node, e.Value.(timerNode)) < 0 {
			pos = e
			break
		}
	}

	if pos != nil {
		slot.InsertBefore(node, pos)
	} else {
		// if slot is empty or @node_ptr is the maximum node
		// in slot, insert it at the last of slot
		slot.PushBack(node)
	}
}

func (w *TimerWheel) deleteTimerNode(node timerNode) {
	var (
		level int
	)

LOOP:
	for level, _ = range w.slot[:] {
		for e, p := w.slot[level].Front(); e != nil; p, e = e, e.Next(p) {
			if e.Value.(timerNode).ID == node.ID {
				w.slot[level].Remove(e)
				// atomic.AddInt64(&w.number, -1)
				break LOOP
			}
		}
	}
}

func (w *TimerWheel) resetTimerNode(node timerNode) {
	var (
		level int
	)

LOOP:
	for level, _ = range w.slot[:] {
		for e, p := w.slot[level].Front(); e != nil; p, e = e, e.Next(p) {
			if e.Value.(timerNode).ID == node.ID {
				n := e.Value.(timerNode)
				n.trig -= n.period
				n.period = node.period
				n.trig += n.period
				w.slot[level].Remove(e)
				w.insertTimerNode(n)
				break LOOP
			}
		}
	}
}

func (w *TimerWheel) deltaDiff(clock int64) int64 {
	var (
		handTime int64
	)

	for idx, hand := range w.hand[:] {
		handTime += hand * msLimit[idx]
	}

	return clock - w.start - handTime
}

func (w *TimerWheel) insertTimerNode(node timerNode) {
	var (
		idx  int
		diff int64
	)

	diff = node.trig - atomic.LoadInt64(&w.clock)
	switch {
	case diff <= 0:
		idx = 0
	case dayNum(diff) != 0:
		idx = 4
	case hourNum(diff) != 0:
		idx = 3
	case minuteNum(diff) != 0:
		idx = 2
	case secondNum(diff) != 0:
		idx = 1
	default:
		idx = 0
	}

	w.insertSlot(idx, node)
}

func (w *TimerWheel) timerCascade(level int) {
	var (
		guard bool
		clock int64
		diff  int64
		cur   timerNode
	)

	clock = atomic.LoadInt64(&w.clock)
	for e, p := w.slot[level].Front(); e != nil; p, e = e, e.Next(p) {
		cur = e.Value.(timerNode)
		diff = cur.trig - clock
		switch {
		case cur.trig <= clock:
			guard = false
		case level == 1:
			guard = secondNum(diff) > 0
		case level == 2:
			guard = minuteNum(diff) > 0
		case level == 3:
			guard = hourNum(diff) > 0
		case level == 4:
			guard = dayNum(diff) > 0
		}

		if guard {
			break
		}

		temp := e
		e, p = p, p.Prev(e)
		w.slot[level].Remove(temp)

		w.insertTimerNode(cur)
	}
}

func (w *TimerWheel) timerUpdate(curTime time.Time) int {
	var (
		clock  int64
		now    int64
		idx    int32
		diff   int64
		maxIdx int32
		inc    [maxTimerLevel + 1]int64
	)

	now = curTime.UnixNano()
	clock = atomic.LoadInt64(&w.clock)
	diff = now - clock
	diff += w.deltaDiff(clock)
	if diff < minDiff*0.7 {
		return -1
	}
	atomic.StoreInt64(&w.clock, now)

	for idx = maxTimerLevel - 1; 0 <= idx; idx-- {
		inc[idx] = diff / msLimit[idx]
		diff %= msLimit[idx]
	}

	maxIdx = 0
	for idx = 0; idx < maxTimerLevel; idx++ {
		if 0 != inc[idx] {
			w.hand[idx] += inc[idx]
			inc[idx+1] += w.hand[idx] / limit[idx]
			w.hand[idx] %= limit[idx]
			maxIdx = idx + 1
		}
	}

	for idx = 1; idx < maxIdx; idx++ {
		w.timerCascade(int(idx))
	}

	return 0
}

func (w *TimerWheel) Stop() {
	w.once.Do(func() {
		close(w.timerQ)
		w.ticker.Stop()
		w.timerQ = nil
	})
}

func (w *TimerWheel) Close() {
	w.Stop()
	w.wg.Wait()
}

////////////////////////////////////////////////
// timer
////////////////////////////////////////////////

type TimerType int32

const (
	eTimerOnce TimerType = 0X1 << 0
	eTimerLoop TimerType = 0X1 << 1
)

// 异步通知timerWheel添加一个timer，有可能失败
func (w *TimerWheel) AddTimer(f TimerFunc, typ TimerType, period int64, arg interface{}) (*Timer, error) {
	if w.timerQ == nil {
		return nil, ErrTimeChannelClosed
	}

	t := &Timer{w: w}
	node := newTimerNode(f, typ, period, arg)
	select {
	case w.timerQ <- timerNodeAction{node: node, action: ADD_TIMER}:
		t.ID = node.ID
		return t, nil
	default:
	}

	return nil, ErrTimeChannelFull
}

func (w *TimerWheel) deleteTimer(t *Timer) error {
	if w.timerQ == nil {
		return ErrTimeChannelClosed
	}

	select {
	case w.timerQ <- timerNodeAction{action: DEL_TIMER, node: timerNode{ID: t.ID}}:
		return nil
	default:
	}

	return ErrTimeChannelFull
}

func (w *TimerWheel) resetTimer(t *Timer, d time.Duration) error {
	if w.timerQ == nil {
		return ErrTimeChannelClosed
	}

	select {
	case w.timerQ <- timerNodeAction{action: RESET_TIMER, node: timerNode{ID: t.ID, period: int64(d)}}:
		return nil
	default:
	}

	return ErrTimeChannelFull
}

func sendTime(_ TimerID, t time.Time, arg interface{}) error {
	select {
	case arg.(chan time.Time) <- t:
	default:
		// gxlog.CInfo("sendTime default")
	}

	return nil
}

func (w *TimerWheel) NewTimer(d time.Duration) *Timer {
	c := make(chan time.Time, 1)
	t := &Timer{
		C: c,
	}

	timer, err := w.AddTimer(sendTime, eTimerOnce, int64(d), c)
	if err == nil {
		t.ID = timer.ID
		t.w = timer.w
		return t
	}

	close(c)
	return nil
}

func (w *TimerWheel) After(d time.Duration) <-chan time.Time {
	//timer := defaultTimer.NewTimer(d)
	//if timer == nil {
	//	return nil
	//}
	//
	//return timer.C
	return w.NewTimer(d).C
}

func goFunc(_ TimerID, _ time.Time, arg interface{}) error {
	go arg.(func())()

	return nil
}

func (w *TimerWheel) AfterFunc(d time.Duration, f func()) *Timer {
	t, _ := w.AddTimer(goFunc, eTimerOnce, int64(d), f)

	return t
}

func (w *TimerWheel) Sleep(d time.Duration) {
	<-w.NewTimer(d).C
}

////////////////////////////////////////////////
// ticker
////////////////////////////////////////////////

func (w *TimerWheel) NewTicker(d time.Duration) *Ticker {
	c := make(chan time.Time, 1)

	timer, err := w.AddTimer(sendTime, eTimerLoop, int64(d), c)
	if err == nil {
		timer.C = c
		return (*Ticker)(timer)
	}

	close(c)
	return nil
}

func (w *TimerWheel) TickFunc(d time.Duration, f func()) *Ticker {
	t, err := w.AddTimer(goFunc, eTimerLoop, int64(d), f)
	if err == nil {
		return (*Ticker)(t)
	}

	return nil
}

func (w *TimerWheel) Tick(d time.Duration) <-chan time.Time {
	return w.NewTicker(d).C
}
