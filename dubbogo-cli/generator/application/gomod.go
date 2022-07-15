package application

const (
	gomodFile = `module dubbo-go-app

go 1.17

require (
	dubbo.apache.org/dubbo-go/v3 v3.0.1
	github.com/dubbogo/grpc-go v1.42.9
	github.com/dubbogo/triple v1.1.8
	github.com/golang/protobuf v1.5.2
	google.golang.org/protobuf v1.27.1
)
`
)

func init() {
	fileMap["gomodFile"] = &fileGenerator{
		path:    ".",
		file:    "go.mod",
		context: gomodFile,
	}
}
