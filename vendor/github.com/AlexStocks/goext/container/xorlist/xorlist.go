// Copyright 2015 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package list implements a xor-doubly linked list in xor style
// whose interface is partially compatible with golang's list.
// incompatible interfaces: XorElement.Prev & XorElement.Next
// & XorList.Front & XorList.Back.
//
// To iterate over a list (where l is a *XorList):
//  var p *XorElement = nil
//	for e := l.Front(); e != nil; p, e = e, e.Next(p) {
//		// do something with e.Value
//	}
// or
// To iterate over a list in reverse (where l is a *XorList):
//  var n *XorElement = nil
//	for e := l.Back(); e != nil; n, e = e, e.Prev(n) {
//		// do something with e.Value
//	}
// or
// To delete a element in iteration
//  var p *XorElement = nil
//	for e := l.Front(); e != nil; p, e = e, e.Next(p) {
//		if condition (e) {
//	      elem := e
//        e, p = p, p.Prev(e)
//        l.Remove(elem)
//	  }
//	}

package gxxorlist

import (
	"fmt"
	"unsafe"
)

// XorElement is an element of a xor-linked list.
type XorElement struct {
	// Compute the bitwise XOR of this element's previous
	// element's address and its next element's address
	// and @PN stores the result
	PN uintptr

	// The value stored with this element.
	Value interface{}
}

func uptr(p *XorElement) uintptr {
	return (uintptr)(unsafe.Pointer(p))
}

func ptr(u uintptr) *XorElement {
	return (*XorElement)(unsafe.Pointer(u))
}

// Next returns the next list element or nil.
func (e *XorElement) Next(prev *XorElement) *XorElement {
	if e == nil || e.PN == 0 {
		return nil
	}
	next := ptr(uptr(prev) ^ e.PN)
	if next != nil && ptr(next.PN) == e { // then next is list's tail
		next = nil
	}

	return next
}

// Prev returns the previous list element or nil.
func (e *XorElement) Prev(next *XorElement) *XorElement {
	if e == nil || e.PN == 0 {
		return nil
	}
	prev := ptr(uptr(next) ^ e.PN)
	if prev != nil && ptr(prev.PN) == e { // then prev is list's head
		prev = nil
	}

	return prev
}

// XorList represents a doubly linked list.
// The zero value for XorList is an empty list ready to use.
type XorList struct {
	head XorElement // first sentinel list element, only &head, head.prev, and head.next are used
	tail XorElement // last sentinel list element, only &tail, tail.prev, and tail.next are used
	len  int        // current list length excluding @list.s two sentinel element
}

// just for test
func (l *XorList) Output() {
	fmt.Printf("fake head{addr:%p, PN:%#x, value:%v} --> \n", &l.head, l.head.PN, l.head.Value)
	for e, p := l.Front(); e != nil; p, e = e, e.Next(p) {
		fmt.Printf("   element{addr:%p, PN:%#x, value:%v} --> \n", &e, e.PN, e.Value)
	}
	fmt.Printf("fake tail{addr:%p, PN:%#x, value:%v}\n", &l.tail, l.tail.PN, l.tail.Value)
}

// Init initializes or clears list l.
func (l *XorList) Init() *XorList {
	l.head.PN = uptr(&l.tail)
	l.tail.PN = uptr(&l.head)
	l.len = 0

	return l
}

// New returns an initialized list.
func New() *XorList { return new(XorList).Init() }

// Len returns the number of elements of list l.
// The complexity is O(1).
func (l *XorList) Len() int { return l.len }

// Front returns the first element of list l or nil.
func (l *XorList) Front() (front, head *XorElement) {
	if l.len == 0 {
		return nil, nil
	}

	return ptr(l.head.PN), &l.head
}

// Back returns the last element of list l or nil.
func (l *XorList) Back() (back, tail *XorElement) {
	if l.len == 0 {
		return nil, nil
	}

	return ptr(l.tail.PN), &l.tail
}

// lazyInit lazily initializes a zero XorList value.
func (l *XorList) lazyInit() {
	if l.head.PN == 0 || l.tail.PN == 0 || ptr(l.head.PN) == &l.tail {
		l.Init()
	}
}

// insert inserts e after @prev and before @next, increments l.len, and returns e.
func (l *XorList) insert(e, prev, next *XorElement) *XorElement {
	e.PN = uptr(prev) ^ uptr(next)
	prev.PN ^= uptr(next) ^ uptr(e)
	next.PN ^= uptr(prev) ^ uptr(e)

	l.len++

	return e
}

// insertValue is a convenience wrapper for insert(&XorElement{Value: v}, prev, next).
func (l *XorList) insertValue(v interface{}, prev, next *XorElement) *XorElement {
	return l.insert(&XorElement{Value: v}, prev, next)
}

// remove removes e from its list, decrements l.len, and returns e.
func (l *XorList) remove(e, prev, next *XorElement) *XorElement {
	prev.PN ^= uptr(e) ^ uptr(next)
	next.PN ^= uptr(e) ^ uptr(prev)
	e.PN = 0

	l.len--

	return e
}

func (l *XorList) prev(e *XorElement) *XorElement {
	prev := &l.head
	cur := prev.Next(nil)
	for cur != nil && cur != e && cur != &l.tail {
		prev, cur = cur, cur.Next(prev)
	}

	if cur != e {
		prev = nil
	}

	return prev
}

func (l *XorList) next(e *XorElement) *XorElement {
	next := &l.tail
	cur := next.Prev(nil)
	for cur != nil && cur != e && cur != &l.head {
		next, cur = cur, cur.Prev(next)
	}

	if cur != e {
		next = nil
	}

	return next
}

// Remove removes e from l if e is an element of list l.
// It returns the element value e.Value.
func (l *XorList) Remove(e *XorElement) interface{} {
	prev := l.prev(e)
	if prev != nil {
		// if e.list == l, l must have been initialized when e was inserted
		// in l or l == nil (e is a zero XorElement) and l.remove will crash
		next := e.Next(prev)
		if next == nil {
			next = &l.tail
		}
		l.remove(e, prev, next)
	}

	return e.Value
}

// PushFront inserts a new element e with value v at the front of list l and returns e.
func (l *XorList) PushFront(v interface{}) *XorElement {
	l.lazyInit()
	return l.insertValue(v, &l.head, ptr(l.head.PN))
}

// PushBack inserts a new element e with value v at the back of list l and returns e.
func (l *XorList) PushBack(v interface{}) *XorElement {
	l.lazyInit()
	return l.insertValue(v, ptr(l.tail.PN), &l.tail)
}

// InsertBefore inserts a new element e with value v immediately before mark and returns e.
// If mark is not an element of l, the list is not modified.
func (l *XorList) InsertBefore(v interface{}, mark *XorElement) *XorElement {
	prev := l.prev(mark)
	if prev == nil {
		return nil
	}

	// see comment in XorList.Remove about initialization of l
	return l.insertValue(v, prev, mark)
}

// InsertAfter inserts a new element e with value v immediately after mark and returns e.
// If mark is not an element of l, the list is not modified.
func (l *XorList) InsertAfter(v interface{}, mark *XorElement) *XorElement {
	next := l.next(mark)
	if next == nil {
		return nil
	}

	// see comment in XorList.Remove about initialization of l
	return l.insertValue(v, mark, next)
}

// MoveToFront moves element e to the front of list l.
// If e is not an element of l, the list is not modified.
func (l *XorList) MoveToFront(e *XorElement) {
	prev := l.prev(e)
	if prev == nil {
		return
	}
	next := e.Next(prev)
	if next == nil {
		next = &l.tail
	}
	e = l.remove(e, prev, next)

	// see comment in XorList.Remove about initialization of l
	l.insert(e, &l.head, ptr(l.head.PN))
}

// MoveToBack moves element e to the back of list l.
// If e is not an element of l, the list is not modified.
func (l *XorList) MoveToBack(e *XorElement) {
	prev := l.prev(e)
	if prev == nil {
		return
	}
	next := e.Next(prev)
	if next == nil {
		next = &l.tail
	}
	e = l.remove(e, prev, next)

	// see comment in XorList.Remove about initialization of l
	l.insert(e, ptr(l.tail.PN), &l.tail)
}

// MoveBefore moves element e to its new position before mark.
// If e or mark is not an element of l, or e == mark, the list is not modified.
func (l *XorList) MoveBefore(e, mark *XorElement) {
	if e == nil || mark == nil || e == mark {
		return
	}

	mark_prev := l.prev(mark)
	if mark_prev == nil {
		return
	}

	e_prev := l.prev(e)
	if e_prev == nil {
		return
	}

	e_next := e.Next(e_prev)
	if e_next == nil {
		e_next = &l.tail
	}
	e = l.remove(e, e_prev, e_next)

	mark_prev = l.prev(mark)
	if mark_prev == nil {
		return
	}
	l.insert(e, mark_prev, mark)
}

// MoveAfter moves element e to its new position after mark.
// If e or mark is not an element of l, or e == mark, the list is not modified.
func (l *XorList) MoveAfter(e, mark *XorElement) {
	if e == nil || mark == nil || e == mark {
		return
	}

	mark_prev := l.prev(mark)
	if mark_prev == nil {
		return
	}

	e_prev := l.prev(e)
	if e_prev == nil {
		return
	}

	e_next := e.Next(e_prev)
	if e_next == nil {
		e_next = &l.tail
	}
	e = l.remove(e, e_prev, e_next)

	mark_next := l.next(mark)
	if mark_next == nil {
		return
	}
	/*
		mark_next = mark.Next(mark_prev)
		if mark_next == nil {
			mark_next = &l.tail
		}
	*/
	l.insert(e, mark, mark_next)
}

// PushBackList inserts a copy of an other list at the back of list l.
// The lists l and other may be the same.
func (l *XorList) PushBackList(other *XorList) {
	l.lazyInit()
	i := other.Len()
	for e, p := other.Front(); i > 0 && e != nil; e, p = e.Next(p), e {
		// l.insertValue(e.Value, l.tail.Prev(nil), &l.tail)
		l.PushBack(e.Value)
		i--
	}
}

// PushFrontList inserts a copy of an other list at the front of list l.
// The lists l and other may be the same.
func (l *XorList) PushFrontList(other *XorList) {
	l.lazyInit()
	i := other.Len()
	for e, n := other.Back(); i > 0 && e != nil; n, e = e, e.Prev(n) {
		// l.insertValue(e.Value, &l.head, (&l.head).Next(nil))
		l.PushFront(e.Value)
		i--
	}
}
