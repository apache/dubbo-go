package application

const (
	configFile = `dubbo:
  registries:
    nacos:
      protocol: nacos
      address: nacos:8848
  protocols:
    triple:
      name: tri
      port: 20000
  provider:
    services:
      GreeterServerImpl:
        interface: "" # read from stub
`
)

func init() {
	fileMap["configFile"] = &fileGenerator{
		path:    "./conf",
		file:    "dubbogo.yaml",
		context: configFile,
	}
}
