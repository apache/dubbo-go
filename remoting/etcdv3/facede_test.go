package etcdv3

import (
	"context"
	"sync"
	"time"

	"github.com/apache/dubbo-go/common"
)

type mockFacade struct {
	client  *Client
	cltLock sync.Mutex
	wg      sync.WaitGroup
	URL     *common.URL
	done    chan struct{}
}

func (r *mockFacade) Client() *Client {
	return r.client
}

func (r *mockFacade) SetClient(client *Client) {
	r.client = client
}

func (r *mockFacade) ClientLock() *sync.Mutex {
	return &r.cltLock
}

func (r *mockFacade) WaitGroup() *sync.WaitGroup {
	return &r.wg
}

func (r *mockFacade) GetDone() chan struct{} {
	return r.done
}

func (r *mockFacade) GetUrl() common.URL {
	return *r.URL
}

func (r *mockFacade) Destroy() {
	close(r.done)
	r.wg.Wait()
}

func (r *mockFacade) RestartCallBack() bool {
	return true
}
func (r *mockFacade) IsAvailable() bool {
	return true
}

func (suite *ClientTestSuite) TestFacade() {

	c := suite.client
	t := suite.T()

	url, err := common.NewURL(context.Background(), "mock://127.0.0.1:2379")
	if err != nil {
		t.Fatal(err)
	}
	mock := &mockFacade{client: c, URL: &url}
	go HandleClientRestart(mock)
	c.Close()

	time.Sleep(2 * time.Second)
}
