package utils

import (
	"github.com/RoaringBitmap/roaring"
	"github.com/apache/dubbo-go/protocol"
)

func JoinIfNotEqual(left *roaring.Bitmap, right *roaring.Bitmap) *roaring.Bitmap {
	if !left.Equals(right) {
		left = left.Clone()
		left.And(right)
	}
	return left
}

func FallbackIfJoinToEmpty(left *roaring.Bitmap, right *roaring.Bitmap) *roaring.Bitmap {
	ret := JoinIfNotEqual(left, right)
	if ret == nil || ret.IsEmpty() {
		return right
	} else {
		return ret
	}
}

func ToIndex(invokers []protocol.Invoker) []int {
	var ret []int
	for i := range invokers {
		ret = append(ret, i)
	}
	return ret
}

func ToBitmap(invokers []protocol.Invoker) *roaring.Bitmap {
	bitmap := roaring.NewBitmap()
	bitmap.AddRange(0, uint64(len(invokers)))
	return bitmap
}
