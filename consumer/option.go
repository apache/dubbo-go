package consumer

type CallOptions struct {
	RequestTimeout string
	Retries        string
}

type CallOption func(*CallOptions)

func WithRequestTimeout(timeout string) CallOption {
	return func(opts *CallOptions) {
		opts.RequestTimeout = timeout
	}
}

func WithRetries(retries string) CallOption {
	return func(opts *CallOptions) {
		opts.Retries = retries
	}
}
