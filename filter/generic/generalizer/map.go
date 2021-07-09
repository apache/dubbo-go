package generalizer

import (
	"reflect"
	"strings"
	"sync"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/mitchellh/mapstructure"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo/hessian2"
)

var (
	mapGeneralizer Generalizer
	once           sync.Once
)

func GetMapGeneralizer() Generalizer {
	once.Do(func() {
		mapGeneralizer = &MapGeneralizer{}
	})
	return mapGeneralizer
}

type MapGeneralizer struct{}

func (g *MapGeneralizer) Generalize(obj interface{}) (gobj interface{}, err error) {
	gobj = objToMap(obj)
	return
}

func (g *MapGeneralizer) Realize(obj interface{}, typ reflect.Type) (interface{}, error) {
	newobj := reflect.New(typ).Interface()
	err := mapstructure.Decode(obj, newobj)
	if err != nil {
		return nil, perrors.Errorf("realizing map failed, %v", err)
	}

	return reflect.ValueOf(newobj).Elem().Interface(), nil
}

func (g *MapGeneralizer) GetType(obj interface{}) (typ string, err error) {
	if typ, err = hessian2.GetJavaName(obj); err != nil && err != hessian2.NilError {
		return
	}

	// TODO: handle nil
	if err == hessian2.NilError {
	}

	return
}

// objToMap converts an object(interface{}) to a map
func objToMap(obj interface{}) interface{} {
	if obj == nil {
		return obj
	}

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	// if obj is a POJO, get the struct from the pointer (if it is a pointer)
	pojo, isPojo := obj.(hessian.POJO)
	if isPojo {
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
			v = v.Elem()
		}
	}

	switch t.Kind() {
	case reflect.Struct:
		result := make(map[string]interface{}, t.NumField())
		if isPojo {
			result["class"] = pojo.JavaClassName()
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)
			kind := value.Kind()
			if !value.CanInterface() {
				logger.Debugf("objToMap for %v is skipped because it couldn't be converted to interface", field)
				continue
			}
			valueIface := value.Interface()
			switch kind {
			case reflect.Ptr:
				if value.IsNil() {
					setInMap(result, field, nil)
					continue
				}
				setInMap(result, field, objToMap(valueIface))
			case reflect.Struct, reflect.Slice, reflect.Map:
				if _, ok := valueIface.(time.Time); ok {
					setInMap(result, field, valueIface)
					continue
				}

				setInMap(result, field, objToMap(valueIface))
			default:
				setInMap(result, field, valueIface)
			}
		}
		return result
	case reflect.Array, reflect.Slice:
		value := reflect.ValueOf(obj)
		newTemps := make([]interface{}, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			newTemp := objToMap(value.Index(i).Interface())
			newTemps = append(newTemps, newTemp)
		}
		return newTemps
	case reflect.Map:
		newTempMap := make(map[interface{}]interface{}, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			if !iter.Value().CanInterface() {
				continue
			}
			key := iter.Key()
			mapV := iter.Value().Interface()
			newTempMap[mapKey(key)] = objToMap(mapV)
		}
		return newTempMap
	case reflect.Ptr:
		return objToMap(v.Elem().Interface())
	default:
		return obj
	}
}

// mapKey converts the map key to interface type
func mapKey(key reflect.Value) interface{} {
	switch key.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8,
		reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Float32,
		reflect.Float64, reflect.String:
		return key.Interface()
	default:
		name := key.String()
		if name == "class" {
			panic(`"class" is a reserved keyword`)
		}
		return name
	}
}

// setInMap sets the struct into the map using the tag or the name of the struct as the key
func setInMap(m map[string]interface{}, structField reflect.StructField, value interface{}) (result map[string]interface{}) {
	result = m
	if tagName := structField.Tag.Get("m"); tagName == "" {
		result[firstLetterToLower(structField.Name)] = value
	} else {
		result[tagName] = value
	}
	return
}

// firstLetterToLower is to lower the first letter
func firstLetterToLower(a string) (b string) {
	b = strings.ToLower(a[:1]) + a[1:]
	return
}
