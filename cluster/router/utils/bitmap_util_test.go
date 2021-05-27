package utils

import (
	"sync"
	"testing"
)

import (
	"github.com/RoaringBitmap/roaring"
	"github.com/stretchr/testify/assert"
)

func TestJoinIfNotEqual(t *testing.T) {
	l := roaring.NewBitmap()
	l.Add(uint32(1))
	l.Add(uint32(2))
	l.Add(uint32(3))
	r := roaring.NewBitmap()
	r.AddRange(1, 4)

	// this is for race condition test
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		JoinIfNotEqual(l, r)
	}()

	go func() {
		defer wg.Done()
		JoinIfNotEqual(l, r)
	}()
	wg.Wait()

	assert.True(t, l.Equals(JoinIfNotEqual(l, r)))
}
