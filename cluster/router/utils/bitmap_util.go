/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"github.com/RoaringBitmap/roaring"
)

import (
	"github.com/apache/dubbo-go/protocol"
)

var EmptyAddr = roaring.NewBitmap()

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
	}
	return ret
}

func ToBitmap(invokers []protocol.Invoker) *roaring.Bitmap {
	bitmap := roaring.NewBitmap()
	bitmap.AddRange(0, uint64(len(invokers)))
	return bitmap
}
