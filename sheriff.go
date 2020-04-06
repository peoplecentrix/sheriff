package sheriff

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var tagName = "groups"

// JSON marshals the object based on groups and wrap with root if specified
func JSON(data interface{}, root string, groups string) interface{} {
	intermediate, err := Marshal(&Options{Groups: strings.Split(groups, ",")}, data)
	if err != nil {
		panic(err)
	}

	if root == "" {
		return intermediate
	}

	return map[string]interface{}{
		root: intermediate,
	}
}

// Options determine which struct fields are being added to the output map.
type Options struct {
	// Groups determine which fields are getting marshalled based on the groups tag.
	// A field with multiple groups (comma-separated) will result in marshalling of that
	// field if one of their groups is specified.
	Groups []string

	// This is used internally so that we can propagate anonymous fields groups tag to all child field.
	nestedGroupsMap map[string][]string
}

// MarshalInvalidTypeError is an error returned to indicate the wrong type has been
// passed to Marshal.
type MarshalInvalidTypeError struct {
	// t reflects the type of the data
	t reflect.Kind
	// data contains the passed data itself
	data interface{}
}

func (e MarshalInvalidTypeError) Error() string {
	return fmt.Sprintf("marshaller: Unable to marshal type %s. Struct required.", e.t)
}

// Marshaller is the interface models have to implement in order to conform to marshalling.
type Marshaller interface {
	Marshal(options *Options) (interface{}, error)
}

// Marshal encodes the passed data into a map which can be used to pass to json.Marshal().
//
// If the passed argument `data` is a struct, the return value will be of type `map[string]interface{}`.
// In all other cases we can't derive the type in a meaningful way and is therefore an `interface{}`.
func Marshal(options *Options, data interface{}) (interface{}, error) {
	v := reflect.ValueOf(data)
	// If data was nil, bail here to avoid panicking. We didn't want to marshal that anyway.
	if !v.IsValid() {
		return nil, nil
	}

	t := v.Type()

	// Initialise nestedGroupsMap,
	// TODO: this may impact the performance, find a better place for this.
	if options.nestedGroupsMap == nil {
		options.nestedGroupsMap = make(map[string][]string)
	}

	if t.Kind() == reflect.Ptr {
		// follow pointer
		t = t.Elem()
	}
	if v.Kind() == reflect.Ptr {
		// follow pointer
		v = v.Elem()
	}

	if t.Kind() != reflect.Struct {
		return marshalValue(options, v)
	}

	dest := make(map[string]interface{})

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		val := v.Field(i)

		jsonTag, jsonOpts := parseTag(field.Tag.Get("json"))

		// If no json tag is provided, use the field Name
		if jsonTag == "" {
			jsonTag = field.Name
		}

		if jsonTag == "-" {
			continue
		}
		if jsonOpts.Contains("omitempty") && isEmptyValue(val) {
			continue
		}
		// skip unexported fields
		if !val.IsValid() || !val.CanInterface() {
			continue
		}

		// if there is an anonymous field which is a struct
		// we want the childs exposed at the toplevel to be
		// consistent with the embedded json marshaller
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// we can skip the group check if if the field is a composition field
		isEmbeddedField := field.Anonymous && val.Kind() == reflect.Struct

		if isEmbeddedField && field.Type.Kind() == reflect.Struct {
			tt := field.Type
			groups := field.Tag.Get(tagName)
			if groups != "" {
				parentGroups := strings.Split(groups, ",")
				for i := 0; i < tt.NumField(); i++ {
					nestedField := tt.Field(i)
					options.nestedGroupsMap[nestedField.Name] = parentGroups
				}
			}
		}

		if !isEmbeddedField {
			var groups []string
			if field.Tag.Get(tagName) != "" {
				groups = strings.Split(field.Tag.Get(tagName), ",")
			}

			if len(groups) == 0 && options.nestedGroupsMap[field.Name] != nil {
				groups = append(groups, options.nestedGroupsMap[field.Name]...)
			}
			shouldShow := len(groups) == 0 || listContains(groups, options.Groups)
			if !shouldShow {
				continue
			}
		}

		v, err := marshalValue(options, val)
		if err != nil {
			return nil, err
		}

		// when a composition field we want to bring the child
		// nodes to the top
		nestedVal, ok := v.(map[string]interface{})
		if isEmbeddedField && ok {
			for key, value := range nestedVal {
				dest[key] = value
			}
		} else {
			dest[jsonTag] = v
		}
	}

	return dest, nil
}

// marshalValue is being used for getting the actual value of a field.
//
// There is support for types implementing the Marshaller interface, arbitrary structs, slices, maps and base types.
func marshalValue(options *Options, v reflect.Value) (interface{}, error) {
	// return nil on nil pointer struct fields
	if !v.IsValid() || !v.CanInterface() {
		return nil, nil
	}
	val := v.Interface()

	if marshaller, ok := val.(Marshaller); ok {
		return marshaller.Marshal(options)
	}
	// types which are e.g. structs, slices or maps and implement one of the following interfaces should not be
	// marshalled by sheriff because they'll be correctly marshalled by json.Marshal instead.
	// Otherwise (e.g. net.IP) a byte slice may be output as a list of uints instead of as an IP string.
	switch val.(type) {
	case json.Marshaler, encoding.TextMarshaler, fmt.Stringer:
		return val, nil
	}
	k := v.Kind()

	if k == reflect.Ptr {
		v = v.Elem()
		val = v.Interface()
		k = v.Kind()
	}

	if k == reflect.Interface || k == reflect.Struct {
		return Marshal(options, val)
	}
	if k == reflect.Slice {
		if v.IsNil() {
			return nil, nil
		}
		l := v.Len()
		dest := make([]interface{}, l)
		for i := 0; i < l; i++ {
			d, err := marshalValue(options, v.Index(i))
			if err != nil {
				return nil, err
			}
			dest[i] = d
		}
		return dest, nil
	}
	if k == reflect.Map {
		if v.IsNil() {
			return nil, nil
		}
		mapKeys := v.MapKeys()
		if len(mapKeys) == 0 {
			dest := make(map[string]interface{})
			return dest, nil
		}
		dest := make(map[string]interface{})
		for _, key := range mapKeys {
			d, err := marshalValue(options, v.MapIndex(key))
			if err != nil {
				return nil, err
			}
			keyString, err := coerceMapKeyToString(key)
			if err != nil {
				return nil, err
			}
			dest[keyString] = d
		}
		return dest, nil
	}
	return val, nil
}

func coerceMapKeyToString(v reflect.Value) (string, error) {
	// Copied from encode.go in the official json package

	if v.Kind() == reflect.String {
		return v.String(), nil
	}

	if tm, ok := v.Interface().(encoding.TextMarshaler); ok {
		buf, err := tm.MarshalText()
		return string(buf), err
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10), nil
	}

	return "", MarshalInvalidTypeError{t: v.Kind(), data: v.Interface()}
}

// contains check if a given key is contained in a slice of strings.
func contains(key string, list []string) bool {
	for _, innerKey := range list {
		if key == innerKey {
			return true
		}
	}
	return false
}

// listContains operates on two string slices and checks if one of the strings in `a`
// is contained in `b`.
func listContains(a []string, b []string) bool {
	for _, key := range a {
		if contains(key, b) {
			return true
		}
	}
	return false
}
