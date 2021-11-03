package capupd

// CapacityUpdater updates capacity evaluator.
// Each method has a stand-alone updater instance, it could be passed by the invocation.
type CapacityUpdater interface {
	// Succeed updates capacity evaluator if the invocation finish successfully.
	Succeed()
	// Failed updates capacity evaluator if the invocation finish unsuccessfully.
	Failed()
}
