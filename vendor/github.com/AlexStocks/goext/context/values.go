// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxcontext provides a key-value context
package gxcontext

import (
	"context"
)

var (
	defaultCtxKey int = 1
)

type Values struct {
	m map[interface{}]interface{}
}

func (v Values) Get(key interface{}) (interface{}, bool) {
	i, b := v.m[key]
	return i, b
}

func (c Values) Set(key interface{}, value interface{}) {
	c.m[key] = value
}

func (c Values) Delete(key interface{}) {
	delete(c.m, key)
}

type ValuesContext struct {
	context.Context
}

func NewValuesContext(ctx context.Context) *ValuesContext {
	if ctx == nil {
		ctx = context.Background()
	}

	return &ValuesContext{
		Context: context.WithValue(
			ctx,
			defaultCtxKey,
			Values{m: make(map[interface{}]interface{})},
		),
	}
}

func (c *ValuesContext) Get(key interface{}) (interface{}, bool) {
	return c.Context.Value(defaultCtxKey).(Values).Get(key)
}

func (c *ValuesContext) Delete(key interface{}) {
	c.Context.Value(defaultCtxKey).(Values).Delete(key)
}

func (c *ValuesContext) Set(key interface{}, value interface{}) {
	c.Context.Value(defaultCtxKey).(Values).Set(key, value)
}
