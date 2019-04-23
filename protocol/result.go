package protocol

type Result interface {
}

/////////////////////////////
// Result Impletment of RPC
/////////////////////////////

type RPCResult struct {
	Err  error
	Rest interface{}
}
