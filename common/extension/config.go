package extension

import (
	"github.com/dubbo/dubbo-go/config"
)

var (
	urls map[string]func(string) (config.ConfigURL, error)
)

func init() {
	// init map
	urls = make(map[string]func(string) (config.ConfigURL, error))
}

func SetURL(name string, v func(string) (config.ConfigURL, error)) {
	urls[name] = v
}

func SetDefaultURLExtension(v func(string) (config.ConfigURL, error)) {
	urls["default"] = v
}

func GetURLExtension(name string, urlString string) (config.ConfigURL, error) {
	if name == "" {
		name = "default"
	}
	return urls[name](urlString)
}
func GetDefaultURLExtension(urlString string) (config.ConfigURL, error) {
	return urls["default"](urlString)
}
