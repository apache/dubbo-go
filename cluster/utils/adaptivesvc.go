package utils

import (
	"fmt"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc"
	adasvcfilter "dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter"
)

var ReachLimitationErrorString = fmt.Sprintf("%s: %s",
	adaptivesvc.ErrAdaptiveSvcInterrupted.Error(),
	adasvcfilter.ErrReachLimitation.Error())

func DoesAdaptiveServiceReachLimitation(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == ReachLimitationErrorString
}

func IsAdaptiveServiceFailed(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), adaptivesvc.ErrAdaptiveSvcInterrupted.Error())
}
