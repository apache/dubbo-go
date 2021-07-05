package hessian2

import (
	"fmt"
	"reflect"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	perrors "github.com/pkg/errors"
)

var (
	NilError = perrors.Errorf("object should not be nil")
	UndeterminedTypeError = perrors.Errorf("object should be a POJO")
)

// GetJavaName returns java name of an object
func GetJavaName(obj interface{}) (string, error) {
	if obj == nil {
		return "", NilError
	}

	t := reflect.TypeOf(obj)
	switch t.Kind() {
	case reflect.Bool:
		return "boolean", nil
	case reflect.Int, reflect.Int64: // in 64-bit processor, Int takes a 64-bit space
		return "long", nil
	case reflect.Int32:
		return "int", nil
	case reflect.Int8, reflect.Int16:
		return "short", nil
	case reflect.Uint, reflect.Uint64: // in 64-bit processor, Uint takes a 64-bit space
		return "unsigned long", nil
	case reflect.Uint32:
		return "unsigned int", nil
	case reflect.Uint16:
		return "unsigned short", nil
	case reflect.Uint8:
		return "char", nil
	case reflect.Float32:
		return "float", nil
	case reflect.Float64:
		return "double", nil
	case reflect.String:
		return "java.lang.String", nil
	case reflect.Array, reflect.Slice:
		ret, err := GetJavaName(reflect.ValueOf(obj).Elem().Interface())
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s[]", ret), nil
	case reflect.Map:
		// TODO: tests for map are required.
		return "java.util.Map", nil
	case reflect.Ptr:
		return GetJavaName(reflect.ValueOf(obj).Elem())
	default:
		pojo, ok := obj.(hessian.POJO)
		if !ok {
			return "", UndeterminedTypeError
		}
		return pojo.JavaClassName(), nil
	}
}

// GetClassDesc get class desc.
// - boolean[].class => "[Z"
// - Object.class => "Ljava/lang/Object;"
func GetClassDesc(v interface{}) string {
	if v == nil {
		return "V"
	}

	switch v := v.(type) {
	// Serialized tags for base types
	case nil:
		return "V"
	case bool:
		return "Z"
	case []bool:
		return "[Z"
	case byte:
		return "B"
	case []byte:
		return "[B"
	case int8:
		return "B"
	case []int8:
		return "[B"
	case int16:
		return "S"
	case []int16:
		return "[S"
	case uint16: // Equivalent to Char of Java
		return "C"
	case []uint16:
		return "[C"
	// case rune:
	//	return "C"
	case int:
		return "J"
	case []int:
		return "[J"
	case int32:
		return "I"
	case []int32:
		return "[I"
	case int64:
		return "J"
	case []int64:
		return "[J"
	case time.Time:
		return "java.util.Date"
	case []time.Time:
		return "[Ljava.util.Date"
	case float32:
		return "F"
	case []float32:
		return "[F"
	case float64:
		return "D"
	case []float64:
		return "[D"
	case string:
		return "java.lang.String"
	case []string:
		return "[Ljava.lang.String;"
	case []hessian.Object:
		return "[Ljava.lang.Object;"
	case map[interface{}]interface{}:
		// return  "java.util.HashMap"
		return "java.util.Map"
	case hessian.POJOEnum:
		return v.(hessian.POJOEnum).JavaClassName()
	//  Serialized tags for complex types
	default:
		t := reflect.TypeOf(v)
		if reflect.Ptr == t.Kind() {
			t = reflect.TypeOf(reflect.ValueOf(v).Elem())
		}
		switch t.Kind() {
		case reflect.Struct:
			return "java.lang.Object"
		case reflect.Slice, reflect.Array:
			if t.Elem().Kind() == reflect.Struct {
				return "[Ljava.lang.Object;"
			}
			// return "java.util.ArrayList"
			return "java.util.List"
		case reflect.Map: // Enter here, map may be map[string]int
			return "java.util.Map"
		default:
			return ""
		}
	}
}
