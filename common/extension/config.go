package extension

import (
	"github.com/dubbo/dubbo-go/config"
)

var (
	url           map[string]func(string) (config.ConfigURL,error)
)

func init() {
	// init map
	url = make(map[string]func(string) (config.ConfigURL,error))
}



func SetURL(name string, v func(string) config.ConfigURL) {
	url[name] = v
}




func GetURLExtension(name string, urlString string) (config.ConfigURL,error){
	if name == "" {
		name = "default"
	}
	return url[name](urlString)
}
func GetDefaultURLExtension(urlString string) (config.ConfigURL,error) {
	return url["default"](urlString)
}

func SetDefaultURLExtension(v func(string) (config.ConfigURL,error)) {
	 url["default"] = v
}
