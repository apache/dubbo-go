package extension

import (
	"context"
	"github.com/dubbo/dubbo-go/cluster"
)

var (
	clusters = make(map[string]func(ctx context.Context) cluster.Cluster)
)

func SetCluster(name string, fcn func(ctx context.Context) cluster.Cluster) {
	clusters[name] = fcn
}

func GetCluster(name string, ctx context.Context) cluster.Cluster {
	return clusters[name](ctx)
}
