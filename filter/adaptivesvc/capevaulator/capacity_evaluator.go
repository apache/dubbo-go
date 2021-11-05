package capevaulator

type CapacityEvaluator interface {
	// EstimatedCapacity is estimated capacity, which reflects the maximum requests handled by the provider.
	EstimatedCapacity() uint64

	// ActualCapacity is actual requests on the provider.
	ActualCapacity() uint64

	// NewCapacityUpdater returns a capacity updater
	NewCapacityUpdater() CapacityUpdater
}
