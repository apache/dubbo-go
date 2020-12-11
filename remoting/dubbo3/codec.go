package dubbo3

type CodeC interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}
