package auth

import (
	"reflect"
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

}

func Test_doSign(t *testing.T) {
	type args struct {
		bytes []byte
		key   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := doSign(tt.args.bytes, tt.args.key); got != tt.want {
				t.Errorf("doSign() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toBytes(t *testing.T) {
	type args struct {
		data []interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toBytes(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("toBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toBytes() got = %v, want %v", got, tt.want)
			}
		})
	}
}
