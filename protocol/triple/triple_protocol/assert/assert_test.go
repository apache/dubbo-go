// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package assert

import (
	"errors"
	"fmt"
	"testing"
)

func TestAssertions(t *testing.T) {
	t.Parallel()

	t.Run("equal", func(t *testing.T) {
		t.Parallel()
		Equal(t, 1, 1, Sprintf("1 == %d", 1))
		NotEqual(t, 1, 2)
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		Nil(t, nil)
		Nil(t, (*chan int)(nil))
		Nil(t, (*func())(nil))
		Nil(t, (*map[int]int)(nil))
		Nil(t, (*pair)(nil))
		Nil(t, (*[]int)(nil))

		NotNil(t, make(chan int))
		NotNil(t, func() {})
		NotNil(t, any(1))
		NotNil(t, make(map[int]int))
		NotNil(t, &pair{})
		NotNil(t, make([]int, 0))

		NotNil(t, "foo")
		NotNil(t, 0)
		NotNil(t, false)
		NotNil(t, pair{})
	})

	t.Run("zero", func(t *testing.T) {
		t.Parallel()
		var n *int
		Zero(t, n)
		var p pair
		Zero(t, p)
		var null *pair
		Zero(t, null)
		var s []int
		Zero(t, s)
		var m map[string]string
		Zero(t, m)
		NotZero(t, 3)
	})

	t.Run("error chain", func(t *testing.T) {
		t.Parallel()
		want := errors.New("base error")
		ErrorIs(t, fmt.Errorf("context: %w", want), want)
	})

	t.Run("unexported fields", func(t *testing.T) {
		t.Parallel()
		// Two pairs differ only in an unexported field.
		p1 := pair{1, 2}
		p2 := pair{1, 3}
		NotEqual(t, p1, p2)
	})

	t.Run("regexp", func(t *testing.T) {
		t.Parallel()
		Match(t, "foobar", `^foo`)
	})

	t.Run("panics", func(t *testing.T) {
		t.Parallel()
		Panics(t, func() { panic("testing") }) //nolint:forbidigo
	})
}

type pair struct {
	First, Second int
}
