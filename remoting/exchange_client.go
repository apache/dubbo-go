package remoting

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	"time"
)

type ExchangeClient struct {
	ConnectTimeout time.Duration
	address        string
}

func newExchangeClient(url common.URL, ConnectTimeout time.Duration, ) *ExchangeClient {
	exchangeClient := &ExchangeClient{
		ConnectTimeout: ConnectTimeout,
		address:        url.Ip + ":" + url.Port,
	}

	return exchangeClient
}

func (client *ExchangeClient) request(invocation *protocol.Invocation, url common.URL, timeout time.Duration,
	result *protocol.RPCResult) error {

}

func (client *ExchangeClient) asyncRequest(invocation *protocol.Invocation, url common.URL, timeout time.Duration,
	callback common.AsyncCallback, result *protocol.RPCResult) error {

}

func (client *ExchangeClient) send(invocation *protocol.Invocation, timeout time.Duration) error {

}
