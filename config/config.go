package config

// Extension - ServiceConfig
type ServiceConfig interface {
	Key() string
	String() string
	ServiceEqual(url URL) bool
	// your service config implements must contain properties below
	Service() string
	Protocol() string
	Version() string
	Group() string
	SetProtocol(string)
	SetService(string)
	SetVersion(string)
	SetGroup(string)
	// export
	Export()
}

// Extension - RegisterConfig
type RegisterConfig interface {
	IsValid() bool
}
