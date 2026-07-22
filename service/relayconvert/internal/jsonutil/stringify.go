package jsonutil

import (
	"fmt"

	"github.com/55gY/new-api-lite/common"
)

func ToJSONString(v interface{}) string {
	bytes, err := common.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(bytes)
}