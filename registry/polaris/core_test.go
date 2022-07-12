package polaris

import (
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestPolarisServiceWatcher_AddSubscriber(t *testing.T) {
	type fields struct {
		consumer       api.ConsumerAPI
		subscribeParam *api.WatchServiceRequest
		lock           *sync.RWMutex
		subscribers    []subscriber
		execOnce       *sync.Once
	}
	type args struct {
		subscriber func(remoting.EventType, []model.Instance)
	}
	var tests []struct {
		name   string
		fields fields
		args   args
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watcher := &PolarisServiceWatcher{
				subscribeParam: &newParam,
				consumer:       newConsumer,
				lock:           &sync.RWMutex{},
				subscribers:    make([]subscriber, 0),
				execOnce:       &sync.Once{},
			}
			assert.Empty(t, watcher)
		})
	}
}
