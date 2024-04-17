go build

for dir in ./test/correctly/*/; do
    cd "$dir" || exit 1

    dir_name=$(basename "$dir")
    go mod tidy
    protoc --go_out=. --go_opt=paths=source_relative --plugin=protoc-gen-go-triple=../../../protoc-gen-go-triple --go-triple_out=. ./proto/greet.proto

    go vet ./proto/*.go
    result=$?

    if [ $result -ne 0 ]; then
        echo "go vet found issues in $dir_name."
        exit $result
    else
        echo "No issues found in $dir_name."
    fi

    cd - || exit 1
done