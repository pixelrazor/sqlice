package sqlice

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Masterminds/squirrel"
)

const fieldNameTag = "db"

type numericOperation int

const (
	opLT numericOperation = iota
	opGT
	opLTOrEQ
	opGTOrEQ
)

// ValueFilterer is the interface that wraps the FilterValue method.
// FilterValue is given a value from slice of elements, and should return true if it is to be included.
type ValueFilterer interface {
	FilterValue(interface{}) bool
}

// ValueFilterFunc is an adapter that allows a function to be used as a custom filter to Filter. It
// implements the squirrel.Sqlizer interface to satisfy requirements, but the implementation of ToSql
// will always return an error
type ValueFilterFunc func(interface{}) bool

// FilterValue calls vff(i)
func (vff ValueFilterFunc) FilterValue(i interface{}) bool {
	return vff(i)
}

// ToSql always returns an error when called
func (vff ValueFilterFunc) ToSql() (string, []interface{}, error) {
	return "", nil, errors.New("ValueFilterFuncs do not implement squirrel.Sqlizer")
}

func Filter(input, output interface{}, filter squirrel.Sqlizer) error {
	inVal, outVal, err := getParamValues(input, output)
	if err != nil {
		return fmt.Errorf("failed to validate in/out params: %w", err)
	}
	// short circuit nil filters
	if filter == nil {
		outVal.Set(inVal)
		return nil
	}

	fields := getFields(inVal.Type().Elem())
	filter, err = sanitizeFilter(filter, fields)
	if err != nil {
		return fmt.Errorf("unable to use filter: %w", err)
	}

	outVal.Set(reflect.MakeSlice(inVal.Type(), 0, 0))
	for i := 0; i < inVal.Len(); i++ {
		val := inVal.Index(i)
		matches, err := matchesFilter(val, filter, fields)
		if err != nil {
			return fmt.Errorf("unable to apply filter: %w", err)
		}
		if matches {
			outVal.Set(reflect.Append(outVal, val))
		}
	}
	return nil
}

func compareValues(v1, v2 reflect.Value, op numericOperation) bool {
	switch reducedKind(v1.Kind()) {
	case reflect.Int64:
		return compareInt(v1, v2, op)
	case reflect.Uint64:
		return compareUint(v1, v2, op)
	case reflect.String:
		return compareString(v1, v2, op)
	case reflect.Float64:
		return compareFloat(v1, v2, op)
	default:
		return false
	}
}

func compareInt(val1, val2 reflect.Value, op numericOperation) bool {
	v1 := val1.Int()
	v2 := val2.Int()
	switch op {
	case opLT:
		return v1 < v2
	case opGT:
		return v1 > v2
	case opLTOrEQ:
		return v1 <= v2
	case opGTOrEQ:
		return v1 >= v2
	default:
		return false
	}
}

func compareUint(val1, val2 reflect.Value, op numericOperation) bool {
	v1 := val1.Uint()
	v2 := val2.Uint()
	switch op {
	case opLT:
		return v1 < v2
	case opGT:
		return v1 > v2
	case opLTOrEQ:
		return v1 <= v2
	case opGTOrEQ:
		return v1 >= v2
	default:
		return false
	}
}

func compareString(val1, val2 reflect.Value, op numericOperation) bool {
	v1 := val1.String()
	v2 := val2.String()
	switch op {
	case opLT:
		return v1 < v2
	case opGT:
		return v1 > v2
	case opLTOrEQ:
		return v1 <= v2
	case opGTOrEQ:
		return v1 >= v2
	default:
		return false
	}
}

func compareFloat(val1, val2 reflect.Value, op numericOperation) bool {
	v1 := val1.Float()
	v2 := val2.Float()
	switch op {
	case opLT:
		return v1 < v2
	case opGT:
		return v1 > v2
	case opLTOrEQ:
		return v1 <= v2
	case opGTOrEQ:
		return v1 >= v2
	default:
		return false
	}
}

func matchesFilter(item reflect.Value, filter squirrel.Sqlizer, fields map[string]fieldInfo) (bool, error) {
	switch filter := filter.(type) {
	case squirrel.And:
		for _, f := range filter {
			matches, err := matchesFilter(item, f, fields)
			if err != nil {
				return false, err
			}
			if !matches {
				return false, nil
			}
		}
		return true, nil
	case squirrel.Or:
		if len(filter) == 0 {
			return true, nil
		}
		for _, f := range filter {
			matches, err := matchesFilter(item, f, fields)
			if err != nil {
				return false, err
			}
			if matches {
				return true, nil
			}
		}
		return false, nil
	case squirrel.Eq:
		for name, value := range filter {
			field := fields[name]
			if !reflect.DeepEqual(item.Field(field.Index).Interface(), value) {
				return false, nil
			}
		}
		return true, nil
	case squirrel.NotEq:
		for name, value := range filter {
			field := fields[name]
			if reflect.DeepEqual(item.Field(field.Index).Interface(), value) {
				return false, nil
			}
		}
		return true, nil
	case squirrel.Gt:
		for name, value := range filter {
			field := fields[name]
			if !compareValues(item.Field(field.Index), reflect.ValueOf(value), opGT) {
				return false, nil
			}
		}
		return true, nil
	case squirrel.Lt:
		for name, value := range filter {
			field := fields[name]
			if !compareValues(item.Field(field.Index), reflect.ValueOf(value), opLT) {
				return false, nil
			}
		}
		return true, nil
	case squirrel.GtOrEq:
		for name, value := range filter {
			field := fields[name]
			if !compareValues(item.Field(field.Index), reflect.ValueOf(value), opGTOrEQ) {
				return false, nil
			}
		}
		return true, nil
	case squirrel.LtOrEq:
		for name, value := range filter {
			field := fields[name]
			if !compareValues(item.Field(field.Index), reflect.ValueOf(value), opLTOrEQ) {
				return false, nil
			}
		}
		return true, nil
	case squirrel.Like:
		panic("not implemented")
	case squirrel.NotLike:
		panic("not implemented")
	case squirrel.ILike:
		panic("not implemented")
	case squirrel.NotILike:
		panic("not implemented")
	case ValueFilterer:
		return filter.FilterValue(item.Interface()), nil
	default:
		return true, nil
	}
}

// sanitizeFilter will convert the filter to lowercase values for field names. It will return an error
// if there's a filtered field that is not present in the struct and if the field and filter types are not
// compatible
func sanitizeFilter(filter squirrel.Sqlizer, fields map[string]fieldInfo) (squirrel.Sqlizer, error) {
	switch filter := filter.(type) {
	case squirrel.And:
		ret, err := sanitizeCond(filter, fields)
		return squirrel.And(ret), err
	case squirrel.Or:
		ret, err := sanitizeCond(filter, fields)
		return squirrel.Or(ret), err
	case squirrel.Eq:
		ret, err := sanitizeMap(filter, fields)
		return squirrel.Eq(ret), err
	case squirrel.NotEq:
		ret, err := sanitizeMap(filter, fields)
		return squirrel.NotEq(ret), err
	case squirrel.Gt:
		ret, err := sanitizeMap(filter, fields)
		return squirrel.Gt(ret), err
	case squirrel.Lt:
		ret, err := sanitizeMap(filter, fields)
		return squirrel.Lt(ret), err
	case squirrel.GtOrEq:
		ret, err := sanitizeMap(filter, fields)
		return squirrel.GtOrEq(ret), err
	case squirrel.LtOrEq:
		ret, err := sanitizeMap(filter, fields)
		return squirrel.LtOrEq(ret), err
	case squirrel.Like:
		ret, err := sanitizeMap(filter, fields)
		return squirrel.Like(ret), err
	case squirrel.NotLike:
		ret, err := sanitizeMap(filter, fields)
		return squirrel.NotLike(ret), err
	case squirrel.ILike:
		ret, err := sanitizeMap(filter, fields)
		return squirrel.ILike(ret), err
	case squirrel.NotILike:
		ret, err := sanitizeMap(filter, fields)
		return squirrel.NotILike(ret), err
	default:
		return filter, nil
	}
}

func sanitizeCond(filters []squirrel.Sqlizer, fields map[string]fieldInfo) ([]squirrel.Sqlizer, error) {
	output := make([]squirrel.Sqlizer, 0, len(filters))
	for _, filter := range filters {
		filter, err := sanitizeFilter(filter, fields)
		if err != nil {
			return nil, err
		}
		output = append(output, filter)
	}
	return output, nil
}

func sanitizeMap(filters map[string]interface{}, fields map[string]fieldInfo) (map[string]interface{}, error) {
	output := make(map[string]interface{})
	for name, value := range filters {
		nameLower := strings.ToLower(name)
		field, ok := fields[nameLower]
		if !ok {
			return nil, fmt.Errorf("struct has no field named '%v'", name)
		}
		expectedKind := reducedKind(field.Type.Kind())
		typesMatch := false
		switch expectedKind {
		case reflect.Int64, reflect.Uint64, reflect.Complex128, reflect.Float64:
			typesMatch = expectedKind == reducedKind(reflect.ValueOf(value).Kind())
		default:
			typesMatch = field.Type == reflect.ValueOf(value).Type()
		}
		if !typesMatch {
			return nil, fmt.Errorf("expected field '%v' to have type %v, got %v", name, field.Type, reflect.ValueOf(value).Type())
		}
		output[nameLower] = value
	}
	return output, nil
}

// reducedKind returns a simplified kind. Numeric kinds are reduced to their biggest representation, as those
// are the forms easily obtainable through a reflect.Value
func reducedKind(kind reflect.Kind) reflect.Kind {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.Int64
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return reflect.Uint64
	case reflect.Float32, reflect.Float64:
		return reflect.Float64
	case reflect.Complex64, reflect.Complex128:
		return reflect.Complex128
	default:
		return kind
	}
}

type fieldInfo struct {
	Index int
	Type  reflect.Type
}

func getFields(t reflect.Type) map[string]fieldInfo {
	fields := make(map[string]fieldInfo)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" { // replace with !IsExported in go 1.17
			continue
		}

		// if the field has the tag, get the name from the tag
		name := field.Name
		if value, ok := field.Tag.Lookup(fieldNameTag); ok {
			name = value
		}
		name = strings.ToLower(name)
		fields[name] = fieldInfo{Index: i, Type: field.Type}
	}
	return fields
}

// getParamValues will check the parameters for the following properties:
// input must be a slice of filterable objects (currently just structs)
// output must be a pointer to a slice of the same type as input
// both input and output should not be nil
// If all conditions are met, the reflect.Value of each is returned. The returned Value for
// output will be addressable.
func getParamValues(input, output interface{}) (reflect.Value, reflect.Value, error) {
	if input == nil {
		return reflect.Value{}, reflect.Value{}, errors.New("input is nil")
	}
	if output == nil {
		return reflect.Value{}, reflect.Value{}, errors.New("output is nil")
	}

	// input checking
	inputValue := reflect.ValueOf(input)
	if inputValue.Kind() != reflect.Slice {
		return reflect.Value{}, reflect.Value{}, errors.New("input is not a slice")
	}
	inputSliceType := inputValue.Type().Elem()
	// TODO: maybe support filtering if it's a type that implements some interface
	if inputSliceType.Kind() != reflect.Struct {
		return reflect.Value{}, reflect.Value{}, errors.New("input slice type is not filter-able")
	}

	// output checking
	outputValue := reflect.ValueOf(output)
	if outputValue.Kind() != reflect.Ptr {
		return reflect.Value{}, reflect.Value{}, errors.New("output is not a valid reference")
	}
	outputValue = outputValue.Elem()
	if outputValue.Kind() != reflect.Slice {
		return reflect.Value{}, reflect.Value{}, errors.New("output is not a reference to a slice")
	}
	outputSliceType := outputValue.Type().Elem()
	if outputSliceType != inputSliceType {
		return reflect.Value{}, reflect.Value{}, errors.New("output slice type is not identical to the input ")
	}
	return inputValue, outputValue, nil
}
