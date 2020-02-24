package auth

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestIsEmpty(t *testing.T) {
	type args struct {
		s          string
		allowSpace bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
		{"test1", args{s: "   ", allowSpace: false}, true},
		{"test2", args{s: "   ", allowSpace: true}, false},
		{"test3", args{s: "hello,dubbo", allowSpace: false}, false},
		{"test4", args{s: "hello,dubbo", allowSpace: true}, false},
		{"test5", args{s: "", allowSpace: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEmpty(tt.args.s, tt.args.allowSpace); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSign(t *testing.T) {
	metadata := "com.ikurento.user.UserProvider::sayHi"
	key := "key"
	signature := Sign(metadata, key)
	assert.NotNil(t, signature)

}

func TestSignWithParams(t *testing.T) {
	metadata := "com.ikurento.user.UserProvider::sayHi"
	key := "key"
	params := []interface{}{
		"a", 1, struct {
			Name string
			Id   int64
		}{"YuYu", 1},
	}
	signature, _ := SignWithParams(params, metadata, key)
	assert.False(t, IsEmpty(signature, false))
}

func Test_doSign(t *testing.T) {
	sign := doSign([]byte("DubboGo"), "key")
	sign1 := doSign([]byte("DubboGo"), "key")
	sign2 := doSign([]byte("DubboGo"), "key2")
	assert.NotNil(t, sign)
	assert.Equal(t, sign1, sign)
	assert.NotEqual(t, sign1, sign2)
}

func Test_toBytes(t *testing.T) {
	params := []interface{}{
		"a", 1, struct {
			Name string
			Id   int64
		}{"YuYu", 1},
	}
	params2 := []interface{}{
		"a", 1, struct {
			Name string
			Id   int64
		}{"YuYu", 1},
	}
	jsonBytes, _ := toBytes(params)
	jsonBytes2, _ := toBytes(params2)
	assert.NotNil(t, jsonBytes)
	assert.Equal(t, jsonBytes, jsonBytes2)
}
