package zookeeper

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	perrors "github.com/pkg/errors"
	"sync"
	"time"
)

type ZkClientContainer interface {
	ZkClient() *ZookeeperClient
	SetZkClient(*ZookeeperClient)
	ZkClientLock() *sync.Mutex
	WaitGroup() *sync.WaitGroup //for wait group control, zk client listener & zk client container
	GetDone() chan struct{}     //for zk client control
	RestartCallBack() bool
	common.Node
}

func HandleClientRestart(r ZkClientContainer) {
	var (
		err error

		failTimes int
	)

	defer r.WaitGroup().Done()
LOOP:
	for {
		select {
		case <-r.GetDone():
			logger.Warnf("(ZkProviderRegistry)reconnectZkRegistry goroutine exit now...")
			break LOOP
			// re-register all services
		case <-r.ZkClient().Done():
			r.ZkClientLock().Lock()
			r.ZkClient().Close()
			zkName := r.ZkClient().name
			zkAddress := r.ZkClient().ZkAddrs
			r.SetZkClient(nil)
			r.ZkClientLock().Unlock()

			// 接zk，直至成功
			failTimes = 0
			for {
				select {
				case <-r.GetDone():
					logger.Warnf("(ZkProviderRegistry)reconnectZkRegistry goroutine exit now...")
					break LOOP
				case <-time.After(time.Duration(1e9 * failTimes * ConnDelay)): // 防止疯狂重连zk
				}
				err = ValidateZookeeperClient(r, WithZkName(zkName))
				logger.Infof("ZkProviderRegistry.validateZookeeperClient(zkAddr{%s}) = error{%#v}",
					zkAddress, perrors.WithStack(err))
				if err == nil {
					if r.RestartCallBack() {
						break
					}
				}
				failTimes++
				if MaxFailTimes <= failTimes {
					failTimes = MaxFailTimes
				}
			}
		}
	}
}
