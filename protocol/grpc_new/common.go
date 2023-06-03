package grpc_new

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	perrors "github.com/pkg/errors"
	"reflect"
)

// ReflectResponse reflect return value
func ReflectResponse(in interface{}, out interface{}) error {
	if in == nil {
		return perrors.Errorf("@in is nil")
	}

	if out == nil {
		return perrors.Errorf("@out is nil")
	}
	if reflect.TypeOf(out).Kind() != reflect.Ptr {
		return perrors.Errorf("@out should be a pointer")
	}

	inValue := hessian.EnsurePackValue(in)
	outValue := hessian.EnsurePackValue(out)

	outType := outValue.Type().String()
	if outType == "interface {}" || outType == "*interface {}" {
		hessian.SetValue(outValue, inValue)
		return nil
	}

	switch inValue.Type().Kind() {
	case reflect.Slice, reflect.Array:
		return CopySlice(inValue, outValue)
	case reflect.Map:
		return CopyMap(inValue, outValue)
	default:
		hessian.SetValue(outValue, inValue)
	}

	return nil
}

// CopySlice copy from inSlice to outSlice
func CopySlice(inSlice, outSlice reflect.Value) error {
	if inSlice.IsNil() {
		return perrors.New("@in is nil")
	}
	if inSlice.Kind() != reflect.Slice {
		return perrors.Errorf("@in is not slice, but %v", inSlice.Kind())
	}

	for outSlice.Kind() == reflect.Ptr {
		outSlice = outSlice.Elem()
	}

	size := inSlice.Len()
	outSlice.Set(reflect.MakeSlice(outSlice.Type(), size, size))

	for i := 0; i < size; i++ {
		inSliceValue := inSlice.Index(i)
		if !inSliceValue.Type().AssignableTo(outSlice.Index(i).Type()) {
			return perrors.Errorf("in element type [%s] can not assign to out element type [%s]",
				inSliceValue.Type().String(), outSlice.Type().String())
		}
		outSlice.Index(i).Set(inSliceValue)
	}

	return nil
}

// CopyMap copy from in map to out map
func CopyMap(inMapValue, outMapValue reflect.Value) error {
	if inMapValue.IsNil() {
		return perrors.New("@in is nil")
	}
	if !inMapValue.CanInterface() {
		return perrors.New("@in's Interface can not be used.")
	}
	if inMapValue.Kind() != reflect.Map {
		return perrors.Errorf("@in is not map, but %v", inMapValue.Kind())
	}

	outMapType := hessian.UnpackPtrType(outMapValue.Type())
	hessian.SetValue(outMapValue, reflect.MakeMap(outMapType))

	outKeyType := outMapType.Key()

	outMapValue = hessian.UnpackPtrValue(outMapValue)
	outValueType := outMapValue.Type().Elem()

	for _, inKey := range inMapValue.MapKeys() {
		inValue := inMapValue.MapIndex(inKey)

		if !inKey.Type().AssignableTo(outKeyType) {
			return perrors.Errorf("in Key:{type:%s, value:%#v} can not assign to out Key:{type:%s} ",
				inKey.Type().String(), inKey, outKeyType.String())
		}
		if !inValue.Type().AssignableTo(outValueType) {
			return perrors.Errorf("in Value:{type:%s, value:%#v} can not assign to out value:{type:%s}",
				inValue.Type().String(), inValue, outValueType.String())
		}
		outMapValue.SetMapIndex(inKey, inValue)
	}

	return nil
}
