package tag

import "github.com/apache/dubbo-go/cluster/router"

type tagRouterRule struct {
	router.BaseRouterRule
	Tags               []tag
	AddressToTagnames  map[string][]string
	TagnameToAddresses map[string][]string
}
type tag struct {
	Name      string
	Addresses string
}
