package schema

import "github.com/getkin/kin-openapi/openapi3"

func allowNumericStringParameter(schema *openapi3.Schema, paramIn string) *openapi3.Schema {
	if schema == nil || schema.Type == nil || !schema.Type.Is("string") {
		return schema
	}

	switch paramIn {
	case "query", "path", "header", "cookie":
	default:
		return schema
	}

	cp := copySchemaScalars(schema)
	cp.Type = nil
	cp.AnyOf = append(cp.AnyOf,
		openapi3.NewStringSchema().NewRef(),
		openapi3.NewFloat64Schema().NewRef(),
	)

	return cp
}
