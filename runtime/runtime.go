package runtime

import (
	"reflect"
)

// IsInterfaceValNil checks whether interface value is of type nil or of value nil or both,
// due to "i == nil" only checks nil interface case.
// refer to https://mangatmodi.medium.com/go-check-nil-interface-the-right-way-d142776edef1.
func IsInterfaceValNil(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}
