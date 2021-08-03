package extension

var (
	configs = map[string]Config{}
)

type Config interface {
	Prefix() string
}

func SetConfig(c Config) {
	configs[c.Prefix()] = c
}
