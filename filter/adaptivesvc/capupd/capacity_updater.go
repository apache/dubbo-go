package capupd

type CapacityUpdater interface {
	// Succeed updates capacity evaluator if the invocation finish successfully.
	Succeed()
	// Failed updates capacity evaluator if the invocation finish unsuccessfully.
	Failed()
}
