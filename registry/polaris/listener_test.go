package polaris

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"
	gxchan "github.com/dubbogo/gost/container/chan"
	"github.com/polarismesh/polaris-go/pkg/model"
	"reflect"
	"testing"
)

func TestNewPolarisListener(t *testing.T) {
	type args struct {
		watcher *PolarisServiceWatcher
	}
	tests := []struct {
		name    string
		args    args
		want    *polarisListener
		wantErr bool
	}{

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPolarisListener(tt.args.watcher)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPolarisListener() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPolarisListener() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateUrl(t *testing.T) {
	type args struct {
		instance model.Instance
	}
	tests := []struct {
		name string
		args args
		want *common.URL
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateUrl(tt.args.instance); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_polarisListener_Close(t *testing.T) {
	type fields struct {
		watcher *PolarisServiceWatcher
		events  *gxchan.UnboundedChan
		closeCh chan struct{}
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func Test_polarisListener_Next(t *testing.T) {
	type fields struct {
		watcher *PolarisServiceWatcher
		events  *gxchan.UnboundedChan
		closeCh chan struct{}
	}
	tests := []struct {
		name    string
		fields  fields
		want    *registry.ServiceEvent
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pl := &polarisListener{
				watcher: tt.fields.watcher,
				events:  tt.fields.events,
				closeCh: tt.fields.closeCh,
			}
			got, err := pl.Next()
			if (err != nil) != tt.wantErr {
				t.Errorf("Next() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Next() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_polarisListener_startListen(t *testing.T) {
	type fields struct {
		watcher *PolarisServiceWatcher
		events  *gxchan.UnboundedChan
		closeCh chan struct{}
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
