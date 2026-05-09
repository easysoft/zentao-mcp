package schema

import (
	"context"
	"encoding/json"

	"github.com/getkin/kin-openapi/openapi3"
)

func (s *Service) buildToolInputSchema(ctx context.Context, doc *openapi3.T, op *openapi3.Operation, pathItem *openapi3.PathItem) json.RawMessage {
	ctx, span := s.tracer.Start(ctx, "BuildToolInputSchema")
	defer span.End()

	s.logger.InfoContext(ctx, "building input schema for tool", "op", op.OperationID)

	schema := openapi3.NewObjectSchema()

	parameters := []*openapi3.ParameterRef{}

	if pathItem != nil {
		parameters = append(parameters, pathItem.Parameters...)
	}

	parameters = append(parameters, op.Parameters...)

	for _, p := range parameters {
		if p == nil || p.Value == nil {
			continue
		}

		if p.Value.Deprecated {
			s.logger.WarnContext(ctx, "skipping deprecated parameter",
				"parameter", p.Value.Name,
				"op", op.OperationID,
			)

			continue
		}

		visited := make(map[*openapi3.Schema]bool)

		if prop := s.resolveSchema(doc, p.Value.Schema, visited); prop != nil {
			prop = allowNumericStringParameter(prop, p.Value.In)

			if p.Value.Description != "" {
				prop.Description = p.Value.Description
			}

			schema.WithProperty(p.Value.Name, prop)

			if p.Value.Required || p.Value.In == "path" {
				schema.Required = append(schema.Required, p.Value.Name)
			}
		}
	}

	if op.RequestBody != nil && op.RequestBody.Value != nil {
		rb := op.RequestBody.Value

		ct, ok := rb.Content["application/json"]
		if ok {
			visited := make(map[*openapi3.Schema]bool)

			if prop := s.resolveSchema(doc, ct.Schema, visited); prop != nil {
				schema.WithProperty("payload", prop)
			}
		}
	}

	b, err := json.Marshal(schema)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal input schema",
			"operation", op.OperationID,
			"error", err,
		)

		return nil
	}

	s.logger.InfoContext(ctx, "tool input schema successfully built",
		"operation", op.OperationID,
		"tags", op.Tags,
	)

	return json.RawMessage(b)
}
