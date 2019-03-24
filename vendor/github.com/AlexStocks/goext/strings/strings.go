// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// 2016/09/28
// Package gxstrings implements string related utilities.
package gxstrings

import (
	"math/rand"
	"time"
	"unicode/utf8"
)

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	src = rand.NewSource(time.Now().UnixNano())
)

// get utf8 character numbers
func StringLength(s string) int {
	return utf8.RuneCountInString(s)
}

// return true if the elem is in the array arr
func Contains(arr []interface{}, elem interface{}) bool {
	for _, e := range arr {
		if e == elem {
			return true
		}
	}

	return false
}

func Strings2Ifs(arr []string) []interface{} {
	iArr := make([]interface{}, 0, len(arr))
	for _, s := range arr {
		iArr = append(iArr, s)
	}

	return iArr
}

func IsSameStringArray(arr1 []string, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}

	for i := range arr1 {
		flag := false
		for j := range arr2 {
			if arr2[j] == arr1[i] {
				flag = true
				break
			}
		}
		if !flag {
			return false
		}
	}

	return true
}

// return true if the @sub is a subset of the @parent
func IsSubset(parent []interface{}, sub []interface{}) bool {
	for i := range sub {
		if !Contains(parent, sub[i]) {
			return false
		}
	}

	return true
}

func Merge(s1 []string, s2 []string) []string {
	// we don't use append because s1 could have extra capacity whose
	// elements would be overwritten, which could cause inadvertent
	// sharing (and race connditions) between concurrent calls
	if len(s1) == 0 {
		return s2
	} else if len(s2) == 0 {
		return s1
	}
	ret := make([]string, len(s1)+len(s2))
	copy(ret, s1)
	copy(ret[len(s1):], s2)
	return ret
}

/*
 code example:
 output:
 // data len: 2 , cap: 3 , data: [one two]
 // data2 len: 2 , cap: 3 , data: [1 3]

 data := []string{"one", "two", "three"}
 ArrayRemoveAt(&data, 2)
 fmt.Println("data len:", len(data), ", cap:", cap(data), ", data:", data)

 data2 := []int{1, 2, 3}
 ArrayRemoveAt(&data2, 1)
 fmt.Println("data2 len:", len(data2), ", cap:", cap(data2), ", data:", data2)
*/
func ArrayRemoveAt(a interface{}, i int) {
	if i < 0 {
		return
	}

	if array, ok := a.(*[]int); ok {
		if len(*array) <= i {
			return
		}
		s := *array
		// s = append(s[:i], s[i+1:]...) // perfectly fine if i is the last element
		// *array = s
		*array = append(s[:i], s[i+1:]...) // perfectly fine if i is the last element
	} else if array, ok := a.(*[]string); ok {
		if len(*array) <= i {
			return
		}
		s := *array
		// s = append(s[:i], s[i+1:]...)
		// *array = s
		*array = append(s[:i], s[i+1:]...)
	}
}

// return return value = @arr1 - @arr2
func Sub(arr1 []string, arr2 []string) []string {
	sub := make([]string, 0, len(arr1))
	for i := range arr1 {
		exist := false
		for j := range arr2 {
			if arr1[i] == arr2[j] {
				exist = true
			}
		}
		if !exist {
			sub = append(sub, arr1[i])
		}
	}

	return sub
}

func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
