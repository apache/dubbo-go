package generic

import (
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/filter/generic/generalizer"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// isCallingToGenericService check if it calls to a generic service
func isCallingToGenericService(invoker protocol.Invoker, invocation protocol.Invocation) bool {
	return isGeneric(invoker.GetURL().GetParam(constant.GENERIC_KEY, "")) &&
		invocation.MethodName() != constant.GENERIC
}

// isMakingAGenericCall check if it is making a generic call to a generic service
func isMakingAGenericCall(invoker protocol.Invoker, invocation protocol.Invocation) bool {
	return isGeneric(invoker.GetURL().GetParam(constant.GENERIC_KEY, "")) &&
		invocation.MethodName() == constant.GENERIC &&
		invocation.Arguments() != nil &&
		len(invocation.Arguments()) == 3
}

// isGeneric receives a generic field from url of invoker to determine whether the service is generic or not
func isGeneric(generic string) bool {
	lowerGeneric := strings.ToLower(generic)
	return lowerGeneric == constant.GenericSerializationDefault ||
		lowerGeneric == constant.GenericSerializationProtobuf
}

// isGenericInvocation determines if the invocation has generic format
func isGenericInvocation(invocation protocol.Invocation) bool {
	return invocation.MethodName() == constant.GENERIC &&
		invocation.Arguments() != nil &&
		len(invocation.Arguments()) == 3
}

// toUnexport is to lower the first letter
func toUnexport(a string) string {
	return strings.ToLower(a[:1]) + a[1:]
}

// toExport is to upper the first letter
func toExport(a string) string {
	return strings.ToUpper(a[:1]) + a[1:]
}

func getGeneralizer(generic string) (g generalizer.Generalizer) {
	switch strings.ToLower(generic) {
	case constant.GenericSerializationDefault:
		g = generalizer.GetMapGeneralizer()
	case constant.GenericSerializationProtobuf:
		panic("implement me")
	default:
		logger.Debugf("\"%s\" is not supported, use the default generalizer(MapGeneralizer)", generic)
		g = generalizer.GetMapGeneralizer()
	}
	return
}
