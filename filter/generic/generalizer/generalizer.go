package generalizer

import "reflect"

type Generalizer interface {

	// Generalize generalizes the object to a general struct.
	// For example:
	// map, the type of the `obj` allows a basic type, e.g. string, and a complicated type which is a POJO, see also
	// `hessian.POJO` at [apache/dubbo-go-hessian2](github.com/apache/dubbo-go-hessian2).
	Generalize(obj interface{}) (interface{}, error)

	// Realize realizes a general struct, described in `obj`, to an object for Golang.
	Realize(obj interface{}, typ reflect.Type) (interface{}, error)

	// GetType returns the type of the `obj`
	GetType(obj interface{}) (string, error)
}
