package sample

const (
	clientConfigFile = `dubbo:
  consumer:
    references:
      GreeterClientImpl:
        protocol: tri
        url: "tri://localhost:20000"
        interface: "" # read from pb`
)

func init() {
	fileMap["clientConfigFile"] = &fileGenerator{
		path:    "./go-client/conf",
		file:    "dubbogo.yaml",
		context: clientConfigFile,
	}
}
