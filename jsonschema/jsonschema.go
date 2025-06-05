package jsonschema

import (
	"github.com/xeipuuv/gojsonschema"
)

func Validate(request any, schema string) (isValid bool, message string, err error) {
	requestLoader := gojsonschema.NewGoLoader(request)

	schemaLoader := gojsonschema.NewStringLoader(schema)

	result, err := gojsonschema.Validate(schemaLoader, requestLoader)
	if err != nil {
		return false, "", err
	}

	if !result.Valid() {
		return false, result.Errors()[0].String(), nil
	}

	return true, "", nil
}
