package extension

import "github.com/apache/dubbo-go/cluster"

var (
	interceptors = make(map[string]func() cluster.Interceptor)
)

// SetClusterInterceptor sets cluster interceptor so that user has chance to inject extra logics before and after
// cluster invoker
func SetClusterInterceptor(name string, fun func() cluster.Interceptor) {
	interceptors[name] = fun
}

// GetClusterInterceptor returns the cluster interceptor instance with the given name
func GetClusterInterceptor(name string) cluster.Interceptor {
	if interceptors[name] == nil {
		panic("cluster_interceptor for " + name + " doesn't exist, make sure the corresponding package is imported")
	}
	return interceptors[name]()
}

// GetClusterInterceptors returns all instances of registered cluster interceptors
func GetClusterInterceptors() []cluster.Interceptor {
	ret := make([]cluster.Interceptor, 0, len(interceptors))
	for _, f := range interceptors {
		ret = append(ret, f())
	}
	return ret
}
