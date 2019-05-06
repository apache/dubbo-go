package extension

import (
	"github.com/dubbo/go-for-apache-dubbo/filter"
)

var (
	filters map[string]func() filter.Filter
)

func init() {
	filters = make(map[string]func() filter.Filter)
}

func SetFilter(name string, v func() filter.Filter) {
	filters[name] = v
}

func GetFilterExtension(name string) filter.Filter {
	return filters[name]()
}
