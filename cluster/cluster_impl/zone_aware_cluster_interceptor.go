package cluster_impl

import (
	"context"
)

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
)

type zoneAwareInterceptor struct {
}

func (z *zoneAwareInterceptor) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	key := constant.REGISTRY_KEY + "." + constant.ZONE_FORCE_KEY
	force := ctx.Value(key)

	if force != nil {
		switch value := force.(type) {
		case bool:
			if value {
				invocation.SetAttachments(key, "true")
			}
		case string:
			if "true" == value {
				invocation.SetAttachments(key, "true")
			}
		default:
			// ignore
		}
	}

	return invoker.Invoke(ctx, invocation)
}

func getZoneAwareInterceptor() cluster.Interceptor {
	return &zoneAwareInterceptor{}
}
