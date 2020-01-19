package rest_client

import (
	"context"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
	"github.com/go-resty/resty/v2"
	"net"
	"net/http"
	"path"
	"time"
)

const (
	RESTY = "resty"
)

func init() {
	extension.SetRestClient(RESTY, GetRestyClient)
}

var restyClient *RestyClient

type RestyClient struct {
	client *resty.Client
}

func NewRestyClient(restOption *rest_interface.RestOptions) *RestyClient {
	if restOption.ConnectTimeout == 0 {
		restOption.ConnectTimeout = 3
	}
	if restOption.RequestTimeout == 0 {
		restOption.RequestTimeout = 3
	}
	client := resty.New()
	client.SetTransport(
		&http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(network, addr, restOption.ConnectTimeout*time.Second)
				if err != nil {
					return nil, err
				}
				err = c.SetDeadline(time.Now().Add(restOption.RequestTimeout * time.Second))
				if err != nil {
					return nil, err
				}
				return c, nil
			},
		})
	return &RestyClient{
		client: client,
	}
}

func (rc *RestyClient) Do(restRequest *rest_interface.RestRequest, res interface{}) error {
	_, err := rc.client.R().
		SetHeader("Content-Type", restRequest.Consumes).
		SetHeader("Accept", restRequest.Produces).
		SetPathParams(restRequest.PathParams).
		SetQueryParams(restRequest.QueryParams).
		SetBody(restRequest.Body).
		SetResult(res).
		Execute(restRequest.Method, "http://"+path.Join(restRequest.Location, restRequest.Path))
	return err
}

func GetRestyClient(restOptions *rest_interface.RestOptions) rest_interface.RestClient {
	return NewRestyClient(restOptions)
}
