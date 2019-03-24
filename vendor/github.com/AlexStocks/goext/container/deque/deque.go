// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// 2017/08/21
// Package gxdeque implements deque(double-eneded queue) in golang.
// ref: https://github.com/juju/utils/blob/master/deque/deque.go
package gxdeque

import "container/list"

// blockLen can be any value above 1. Raising the blockLen decreases
// the average number of memory allocations per item, but increases
// the amount of memory "wasted". Raising blockLen also doesn't
// necessarily make Deque operations faster. 64 is used by Python's
// deque and seems to be a sweet spot on the author's machine too.
const (
	blockLen    = 64
	blockCenter = (blockLen - 1) / 2
)

// Deque implements an efficient double-ended queue.
//
// Internally it is composed of a doubly-linked list (list.List) of
// blocks. Each block is a slice that holds 0 to blockLen items. The
// Deque starts with one block. Blocks are added to the front and back
// when the edge blocks fill up as items are pushed onto the
// deque. Edge blocks are removed when blocks become empty as items
// are popped off the Deque.
//
// Only the front and back blocks may contain less than blockLen items.
// The first element in the Deque is d.blocks.Front().Value[d.frontIdx].
// The last element in the Deque is d.blocks.Back().Value[d.backIdx].
//
// This approach is more efficient than using a standard doubly-linked
// list for a queue because memory allocation is only required every
// blockLen items, instead of for each pushed item. Conversely, fewer
// memory deallocations are required when popping items. Bookkeeping
// overhead per item is also reduced.
type Deque struct {
	maxLen            int       // maximum element number of @Deque
	blocks            list.List // blockT base
	frontIdx, backIdx int       // the first and the last emement
	len               int       // current element number
}

type blockT []interface{}

// New returns a new Deque instance.
func New() *Deque {
	return NewWithMaxLen(0)
}

// New returns a new Deque instance which is limited to a certain
// length. Pushes which cause the length to exceed the specified size
// will cause an item to be dropped from the opposing side.
//
// A maxLen of 0 means that there is no maximum length limit in place.
func NewWithMaxLen(maxLen int) *Deque {
	d := Deque{maxLen: maxLen}
	d.blocks.PushBack(newBlock())
	d.recenter()
	return &d
}

func newBlock() blockT {
	return make(blockT, blockLen)
}

func (d *Deque) recenter() {
	// The indexes start crossed at the middle of the block so that
	// the first push on either side has both indexes pointing at the
	// first item.
	d.frontIdx = blockCenter + 1
	d.backIdx = blockCenter
}

// Len returns the number of items stored in the queue.
func (d *Deque) Len() int {
	return d.len
}

// PushBack adds an item to the back of the queue.
func (d *Deque) PushBack(item interface{}) {
	var block blockT
	if d.backIdx == blockLen-1 {
		// The current back block is full so add another.
		block = newBlock()
		d.blocks.PushBack(block)
		d.backIdx = -1
	} else {
		block = d.blocks.Back().Value.(blockT)
	}

	d.backIdx++
	block[d.backIdx] = item
	d.len++

	if d.maxLen > 0 && d.len > d.maxLen {
		d.PopFront()
	}
}

// PushFront adds an item to the front of the queue.
func (d *Deque) PushFront(item interface{}) {
	var block blockT
	if d.frontIdx == 0 {
		// The current front block is full so add another.
		block = newBlock()
		d.blocks.PushFront(block)
		d.frontIdx = blockLen
	} else {
		block = d.blocks.Front().Value.(blockT)
	}

	d.frontIdx--
	block[d.frontIdx] = item
	d.len++

	if d.maxLen > 0 && d.len > d.maxLen {
		d.PopBack()
	}
}

// PopBack removes an item from the back of the queue and returns
// it. The returned flag is true unless there were no items left in
// the queue.
func (d *Deque) PopBack() (interface{}, bool) {
	if d.len < 1 {
		return nil, false
	}

	elem := d.blocks.Back()
	block := elem.Value.(blockT)
	item := block[d.backIdx]
	block[d.backIdx] = nil
	d.backIdx--
	d.len--

	if d.backIdx == -1 {
		// The back block is now empty.
		if d.len == 0 {
			d.recenter() // Deque is empty so reset.
		} else {
			d.blocks.Remove(elem)
			d.backIdx = blockLen - 1
		}
	}

	return item, true
}

// PopFront removes an item from the front of the queue and returns
// it. The returned flag is true unless there were no items left in
// the queue.
func (d *Deque) PopFront() (interface{}, bool) {
	if d.len < 1 {
		return nil, false
	}

	elem := d.blocks.Front()
	block := elem.Value.(blockT)
	item := block[d.frontIdx]
	block[d.frontIdx] = nil
	d.frontIdx++
	d.len--

	if d.frontIdx == blockLen {
		// The front block is now empty.
		if d.len == 0 {
			d.recenter() // Deque is empty so reset.
		} else {
			d.blocks.Remove(elem)
			d.frontIdx = 0
		}
	}

	return item, true
}

// PeekBack gets an item from the back of the queue and returns
// it. The returned flag is true unless there were no items left in
// the queue.
func (d *Deque) PeekBack() (interface{}, bool) {
	if d.len < 1 {
		return nil, false
	}

	elem := d.blocks.Back()
	block := elem.Value.(blockT)
	item := block[d.backIdx]

	return item, true
}

// PeekFront gets an item from the front of the queue and returns
// it. The returned flag is true unless there were no items left in
// the queue.
func (d *Deque) PeekFront() (interface{}, bool) {
	if d.len < 1 {
		return nil, false
	}

	elem := d.blocks.Front()
	block := elem.Value.(blockT)
	item := block[d.frontIdx]

	return item, true
}
