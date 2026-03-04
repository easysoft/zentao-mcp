package schema

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/getkin/kin-openapi/openapi3"
)

func (s *Service) buildToolOutputSchema(ctx context.Context, doc *openapi3.T, op *openapi3.Operation) (json.RawMessage, bool) {
	ctx, span := s.tracer.Start(ctx, "BuildToolOutputSchema")
	defer span.End()

	s.logger.InfoContext(ctx, "building output schema for tool", "op", op.OperationID)

	if op.Responses == nil {
		s.logger.WarnContext(ctx, "no responses defined for operation", "op", op.OperationID)

		return nil, false
	}

	responseRef := findSuccessResponse(op.Responses)
	if responseRef == nil || responseRef.Value == nil {
		return nil, false
	}

	resp := responseRef.Value

	jsonContent, ok := resp.Content["application/json"]
	if !ok || jsonContent.Schema == nil {
		return nil, false
	}

	visited := make(map[*openapi3.Schema]bool)

	schema := s.resolveSchema(doc, jsonContent.Schema, visited)
	if schema == nil {
		return nil, false
	}

	var wrapped bool

	if schema.Type == nil || !schema.Type.Is("object") {
		schema = &openapi3.Schema{
			Type: &openapi3.Types{"object"},
			Properties: openapi3.Schemas{
				"result": &openapi3.SchemaRef{Value: schema},
			},
			Required: []string{"result"},
		}

		wrapped = true
	}

	b, err := json.Marshal(schema)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal output schema",
			"operation", op.OperationID,
			"error", err,
		)

		return nil, false
	}

	s.logger.InfoContext(ctx, "tool output schema successfully built", "operation", op.OperationID)

	return json.RawMessage(b), wrapped
}

func findSuccessResponse(responses *openapi3.Responses) *openapi3.ResponseRef {
	for _, code := range []int{200, 201, 202, 204} {
		if ref := responses.Status(code); ref != nil {
			return ref
		}
	}

	respMap := responses.Map()
	keys := make([]string, 0, len(respMap))

	for k := range respMap {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		if len(k) == 3 && k[0] == '2' {
			return respMap[k]
		}
	}

	return responses.Default()
}
