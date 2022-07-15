package sample

const (
	serverConfigFile = `dubbo:
  protocols:
    triple:
      name: tri
      port: 20000
  provider:
    services:
      GreeterProvider:
        interface: "" # read from pb`
)

func init() {
	fileMap["srvConfGenerator"] = &fileGenerator{
		path:    "./go-server/conf",
		file:    "dubbogo.yaml",
		context: serverConfigFile,
	}
}
