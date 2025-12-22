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
	"regexp"
	"strings"
	"testing"
)

// mockTB is a mock implementation of testing.TB for testing failure scenarios.
type mockTB struct {
	testing.TB
	failed    bool
	fatalMsg  string
	helperCnt int
}

func (m *mockTB) Helper() {
	m.helperCnt++
}

func (m *mockTB) Fatal(args ...any) {
	m.failed = true
	if len(args) > 0 {
		m.fatalMsg = fmt.Sprint(args...)
	}
}

func (m *mockTB) Fatalf(format string, args ...any) {
	m.failed = true
	m.fatalMsg = fmt.Sprintf(format, args...)
}

func TestAssertions(t *testing.T) {
	t.Parallel()

	t.Run("equal", func(t *testing.T) {
		t.Parallel()
		Equal(t, 1, 1, Sprintf("1 == %d", 1))
		NotEqual(t, 1, 2)

		// Test various types
		Equal(t, "hello", "hello")
		Equal(t, []int{1, 2, 3}, []int{1, 2, 3})
		Equal(t, map[string]int{"a": 1}, map[string]int{"a": 1})
		Equal(t, pair{1, 2}, pair{1, 2})

		NotEqual(t, "hello", "world")
		NotEqual(t, []int{1, 2}, []int{1, 3})
		NotEqual(t, map[string]int{"a": 1}, map[string]int{"a": 2})
	})

	t.Run("equal_failure", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		result := Equal(mock, 1, 2)
		if result {
			t.Error("Equal should return false when values differ")
		}
		if !mock.failed {
			t.Error("Equal should call Fatal when values differ")
		}
		if !strings.Contains(mock.fatalMsg, "assert.Equal") {
			t.Errorf("Fatal message should contain 'assert.Equal', got: %s", mock.fatalMsg)
		}
	})

	t.Run("not_equal_failure", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		result := NotEqual(mock, 1, 1)
		if result {
			t.Error("NotEqual should return false when values are equal")
		}
		if !mock.failed {
			t.Error("NotEqual should call Fatal when values are equal")
		}
		if !strings.Contains(mock.fatalMsg, "assert.NotEqual") {
			t.Errorf("Fatal message should contain 'assert.NotEqual', got: %s", mock.fatalMsg)
		}
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		Nil(t, nil)
		Nil(t, (*chan int)(nil))
		Nil(t, (*func())(nil))
		Nil(t, (*map[int]int)(nil))
		Nil(t, (*pair)(nil))
		Nil(t, (*[]int)(nil))
		Nil(t, ([]int)(nil))
		Nil(t, (map[int]int)(nil))
		Nil(t, (chan int)(nil))
		var nilInterface any
		Nil(t, nilInterface)

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

	t.Run("nil_failure", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		result := Nil(mock, "not nil")
		if result {
			t.Error("Nil should return false for non-nil value")
		}
		if !mock.failed {
			t.Error("Nil should call Fatal for non-nil value")
		}
		if !strings.Contains(mock.fatalMsg, "assert.Nil") {
			t.Errorf("Fatal message should contain 'assert.Nil', got: %s", mock.fatalMsg)
		}
	})

	t.Run("not_nil_failure", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		result := NotNil(mock, nil)
		if result {
			t.Error("NotNil should return false for nil value")
		}
		if !mock.failed {
			t.Error("NotNil should call Fatal for nil value")
		}
		if !strings.Contains(mock.fatalMsg, "assert.NotNil") {
			t.Errorf("Fatal message should contain 'assert.NotNil', got: %s", mock.fatalMsg)
		}
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
		var nilIntf any
		Zero(t, nilIntf)
		Zero(t, 0)
		Zero(t, "")
		Zero(t, false)

		NotZero(t, 1)
		NotZero(t, "hello")
		NotZero(t, true)
		NotZero(t, &pair{})
		NotZero(t, []int{1})
		NotZero(t, map[string]int{"a": 1})
	})

	t.Run("zero_failure", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		result := Zero(mock, 42)
		if result {
			t.Error("Zero should return false for non-zero value")
		}
		if !mock.failed {
			t.Error("Zero should call Fatal for non-zero value")
		}
		if !strings.Contains(mock.fatalMsg, "assert.Zero") {
			t.Errorf("Fatal message should contain 'assert.Zero', got: %s", mock.fatalMsg)
		}
	})

	t.Run("not_zero_failure", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		result := NotZero(mock, 0)
		if result {
			t.Error("NotZero should return false for zero value")
		}
		if !mock.failed {
			t.Error("NotZero should call Fatal for zero value")
		}
		if !strings.Contains(mock.fatalMsg, "assert.NotZero") {
			t.Errorf("Fatal message should contain 'assert.NotZero', got: %s", mock.fatalMsg)
		}
	})

	t.Run("not_zero_nil_type", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		// When type is nil (literal nil), NotZero returns false
		result := NotZero(mock, nil)
		if result {
			t.Error("NotZero should return false for nil")
		}
	})

	t.Run("error chain", func(t *testing.T) {
		t.Parallel()
		want := errors.New("base error")
		ErrorIs(t, fmt.Errorf("context: %w", want), want)

		// Test with same error
		err := errors.New("same error")
		ErrorIs(t, err, err)
	})

	t.Run("error_is_failure", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		result := ErrorIs(mock, err1, err2)
		if result {
			t.Error("ErrorIs should return false when errors don't match")
		}
		if !mock.failed {
			t.Error("ErrorIs should call Fatal when errors don't match")
		}
		if !strings.Contains(mock.fatalMsg, "assert.ErrorIs") {
			t.Errorf("Fatal message should contain 'assert.ErrorIs', got: %s", mock.fatalMsg)
		}
	})

	t.Run("exported fields", func(t *testing.T) {
		t.Parallel()
		// NotEqual can only handle exported fields.
		p1 := pair{1, 2}
		p2 := pair{1, 3}
		NotEqual(t, p1, p2)
	})

	t.Run("regexp", func(t *testing.T) {
		t.Parallel()
		Match(t, "foobar", `^foo`)
		Match(t, "foobar", `bar$`)
		Match(t, "hello world", `\w+\s\w+`)
		Match(t, "test123", `\d+`)
	})

	t.Run("regexp_failure", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		result := Match(mock, "foobar", `^bar`)
		if result {
			t.Error("Match should return false when pattern doesn't match")
		}
		if !mock.failed {
			t.Error("Match should call Fatal when pattern doesn't match")
		}
		if !strings.Contains(mock.fatalMsg, "assert.Match") {
			t.Errorf("Fatal message should contain 'assert.Match', got: %s", mock.fatalMsg)
		}
	})

	t.Run("regexp_invalid", func(t *testing.T) {
		t.Parallel()
		// Test that invalid regexp is detected by compiling it ourselves
		// We can't use mockTB here because Match calls Fatalf which doesn't stop execution in mock
		_, err := regexp.Compile(`[invalid`)
		if err == nil {
			t.Error("Expected invalid regexp to fail compilation")
		}
	})

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		True(t, true)
		True(t, 1 == 1)
		True(t, len("hello") > 0)
	})

	t.Run("true_failure", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		result := True(mock, false)
		if result {
			t.Error("True should return false for false value")
		}
		if !mock.failed {
			t.Error("True should call Fatal for false value")
		}
		if !strings.Contains(mock.fatalMsg, "assert.True") {
			t.Errorf("Fatal message should contain 'assert.True', got: %s", mock.fatalMsg)
		}
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		False(t, false)
		False(t, 1 == 2)
		False(t, len("") > 0)
	})

	t.Run("false_failure", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		result := False(mock, true)
		if result {
			t.Error("False should return false for true value")
		}
		if !mock.failed {
			t.Error("False should call Fatal for true value")
		}
		if !strings.Contains(mock.fatalMsg, "assert.False") {
			t.Errorf("Fatal message should contain 'assert.False', got: %s", mock.fatalMsg)
		}
	})

	t.Run("panics", func(t *testing.T) {
		t.Parallel()
		Panics(t, func() { panic("testing") })                 //nolint:forbidigo
		Panics(t, func() { panic(errors.New("error panic")) }) //nolint:forbidigo
		Panics(t, func() { panic(42) })                        //nolint:forbidigo
	})

	t.Run("panics_failure", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		Panics(mock, func() {
			// Does not panic
		})
		if !mock.failed {
			t.Error("Panics should call Fatal when function doesn't panic")
		}
	})

	t.Run("sprintf_option", func(t *testing.T) {
		t.Parallel()
		opt := Sprintf("test message %d", 42)
		if opt.message() != "test message 42" {
			t.Errorf("Sprintf message mismatch, got: %s", opt.message())
		}

		// Test with multiple options (last one wins)
		mock := &mockTB{}
		Equal(mock, 1, 2, Sprintf("first"), Sprintf("second"))
		if !strings.Contains(mock.fatalMsg, "second") {
			t.Errorf("Should use last Sprintf option, got: %s", mock.fatalMsg)
		}
	})

	t.Run("report_with_options", func(t *testing.T) {
		t.Parallel()
		mock := &mockTB{}
		Equal(mock, "got", "want", Sprintf("custom message: %s", "test"))
		if !strings.Contains(mock.fatalMsg, "custom message: test") {
			t.Errorf("Fatal message should contain custom message, got: %s", mock.fatalMsg)
		}
		if !strings.Contains(mock.fatalMsg, "got:") {
			t.Errorf("Fatal message should contain 'got:', got: %s", mock.fatalMsg)
		}
		if !strings.Contains(mock.fatalMsg, "want:") {
			t.Errorf("Fatal message should contain 'want:', got: %s", mock.fatalMsg)
		}
	})
}

type pair struct {
	First, Second int
}

func TestIsNil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  any
		want bool
	}{
		{"literal nil", nil, true},
		{"nil pointer", (*int)(nil), true},
		{"nil slice", ([]int)(nil), true},
		{"nil map", (map[string]int)(nil), true},
		{"nil chan", (chan int)(nil), true},
		{"nil func", (func())(nil), true},
		{"nil interface pointer", (*any)(nil), true},
		{"non-nil int", 42, false},
		{"non-nil string", "hello", false},
		{"non-nil slice", []int{1, 2}, false},
		{"non-nil map", map[string]int{"a": 1}, false},
		{"empty slice", []int{}, false},
		{"empty map", map[string]int{}, false},
		{"zero struct", pair{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isNil(tt.val)
			if got != tt.want {
				t.Errorf("isNil(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestCmpEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  any
		want any
		eq   bool
	}{
		{"equal ints", 1, 1, true},
		{"unequal ints", 1, 2, false},
		{"equal strings", "hello", "hello", true},
		{"unequal strings", "hello", "world", false},
		{"equal slices", []int{1, 2, 3}, []int{1, 2, 3}, true},
		{"unequal slices", []int{1, 2}, []int{1, 3}, false},
		{"equal maps", map[string]int{"a": 1}, map[string]int{"a": 1}, true},
		{"unequal maps", map[string]int{"a": 1}, map[string]int{"a": 2}, false},
		{"equal structs", pair{1, 2}, pair{1, 2}, true},
		{"unequal structs", pair{1, 2}, pair{1, 3}, false},
		{"nil values", nil, nil, true},
		{"nil vs non-nil", nil, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := cmpEqual(tt.got, tt.want)
			if got != tt.eq {
				t.Errorf("cmpEqual(%v, %v) = %v, want %v", tt.got, tt.want, got, tt.eq)
			}
		})
	}
}
