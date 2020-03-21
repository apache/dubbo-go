package impl

type Serializer interface {
	Marshal(p DubboPackage) ([]byte, error)
	Unmarshal([]byte, *DubboPackage) error
}
