package rest_interface

import (
	"time"
)

type RestOptions struct {
	RequestTimeout time.Duration
	ConnectTimeout time.Duration
}

type RestRequest struct {
	Location    string
	Path        string
	Produces    string
	Consumes    string
	Method      string
	PathParams  map[string]string
	QueryParams map[string]string
	Body        map[string]interface{}
	Headers     map[string]string
}

type RestClient interface {
	Do(request *RestRequest, res interface{}) error
}
