package application

const (
	gitignoreFile = `./build/!Dockerfile
`
)

func init() {
	fileMap["gitignoreFile"] = &fileGenerator{
		path:    ".",
		file:    ".gitignore",
		context: gitignoreFile,
	}
}
