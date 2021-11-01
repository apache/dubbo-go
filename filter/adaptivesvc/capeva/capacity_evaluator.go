package capeva

import "dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/capupd"

type CapacityEvaluator interface {
	// Estimated is estimated capacity, which reflects the maximum requests handled by the provider.
	Estimated() int64
	UpdateEstimated(value int64)

	// Actual is actual requests on the provider.
	Actual() int64
	// UpdateActual updates by delta, for instance, if a new request comes, the delta should be 1.
	UpdateActual(delta int64)

	// NewCapacityUpdater returns a capacity updater
	NewCapacityUpdater() capupd.CapacityUpdater
}