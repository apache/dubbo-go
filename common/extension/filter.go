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
	if filters[name] == nil {
		panic("filter for " + name + " is not existing, you must import corresponding package.")
	}
	return filters[name]()
}
