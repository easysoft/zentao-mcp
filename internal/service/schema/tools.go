package schema

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/merzzzl/openapi-mcp-server/internal/models"
)

// Tools extracts all API operations as tool definitions.
func (s *Service) Tools(ctx context.Context) ([]models.ToolDefinition, error) {
	ctx, span := s.tracer.Start(ctx, "Tools")
	defer span.End()

	s.logger.InfoContext(ctx, "building tools")

	doc, err := s.loader.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}

	s.metrics.SchemaLoads.Add(ctx, 1)

	pathsMap := doc.Paths.Map()
	pathKeys := make([]string, 0, len(pathsMap))

	for p := range pathsMap {
		pathKeys = append(pathKeys, p)
	}

	sort.Strings(pathKeys)

	var tools []models.ToolDefinition

	for _, p := range pathKeys {
		pi := pathsMap[p]
		if pi == nil {
			continue
		}

		tools = append(tools, s.pathTools(ctx, doc, p, pi)...)
	}

	s.logger.InfoContext(ctx, "tools built", "count", len(tools))

	return tools, nil
}

func (s *Service) pathTools(ctx context.Context, doc *openapi3.T, p string, pi *openapi3.PathItem) []models.ToolDefinition {
	ordered := []struct {
		verb string
		op   *openapi3.Operation
	}{
		{"GET", pi.Get},
		{"POST", pi.Post},
		{"PUT", pi.Put},
		{"PATCH", pi.Patch},
		{"DELETE", pi.Delete},
		{"HEAD", pi.Head},
		{"OPTIONS", pi.Options},
	}

	var tools []models.ToolDefinition

	for _, m := range ordered {
		if m.op == nil {
			continue
		}

		td := s.buildTool(ctx, doc, p, m.verb, m.op, pi)
		tools = append(tools, td)
	}

	return tools
}

func (s *Service) buildTool(ctx context.Context, doc *openapi3.T, p, method string, op *openapi3.Operation, pi *openapi3.PathItem) models.ToolDefinition {
	opID := op.OperationID
	if opID == "" {
		opID = strings.ToLower(method) + "_" + sanitizePathName(p)

		s.logger.WarnContext(ctx, "no operation id, using generated",
			"generated_id", opID,
			"method", method,
			"path", p,
		)
	}

	desc := strings.TrimSpace(firstNonEmpty(
		op.Summary, op.Description, fmt.Sprintf("%s %s", method, p), opID,
	))

	input := s.buildToolInputSchema(ctx, doc, op, pi)
	output, wrapOutput := s.buildToolOutputSchema(ctx, doc, op)
	params := collectParams(op, pi)

	hasBody := op.RequestBody != nil && op.RequestBody.Value != nil

	return models.ToolDefinition{
		OperationID:  opID,
		Method:       method,
		Path:         p,
		Description:  desc,
		InputSchema:  input,
		OutputSchema: output,
		Params:       params,
		HasBody:      hasBody,
		WrapOutput:   wrapOutput,
	}
}

func collectParams(op *openapi3.Operation, pi *openapi3.PathItem) []models.ToolParam {
	var refs openapi3.Parameters

	if pi != nil {
		refs = append(refs, pi.Parameters...)
	}

	refs = append(refs, op.Parameters...)

	var params []models.ToolParam

	for _, ref := range refs {
		if ref == nil || ref.Value == nil {
			continue
		}

		params = append(params, models.ToolParam{
			Name: ref.Value.Name,
			In:   ref.Value.In,
		})
	}

	return params
}

func sanitizePathName(p string) string {
	s := strings.Trim(p, "/")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "{", "")
	s = strings.ReplaceAll(s, "}", "")
	s = strings.ReplaceAll(s, "-", "_")

	return s
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}

	return ""
}
