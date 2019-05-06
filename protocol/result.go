package protocol

type Result interface {
	Error() error
	Result() interface{}
}

/////////////////////////////
// Result Impletment of RPC
/////////////////////////////

type RPCResult struct {
	Err  error
	Rest interface{}
}

func (r *RPCResult) Error() error {
	return r.Err
}

func (r *RPCResult) Result() interface{} {
	return r.Rest
}
