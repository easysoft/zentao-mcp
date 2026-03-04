package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/merzzzl/openapi-mcp-server/internal/models"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	toon "github.com/toon-format/toon-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RegisterAllTools registers all OpenAPI operations as MCP tools.
func (c *Controller) RegisterAllTools(ctx context.Context, server *mcpsdk.Server) error {
	ctx, span := c.tracer.Start(ctx, "RegisterAllTools")
	defer span.End()

	c.logger.InfoContext(ctx, "registering tools")

	tools, err := c.schema.Tools(ctx)
	if err != nil {
		return fmt.Errorf("get tools: %w", err)
	}

	for i := range tools {
		td := &tools[i]

		if !c.operationChecker(td.Method, td.Path) {
			continue
		}

		c.registerTool(ctx, server, td)
	}

	c.logger.InfoContext(ctx, "tools registered")

	return nil
}

func (c *Controller) registerTool(ctx context.Context, server *mcpsdk.Server, td *models.ToolDefinition) {
	c.logger.InfoContext(ctx, "adding tool",
		"tool", td.OperationID,
		"method", td.Method,
		"path", td.Path,
	)

	t := &mcpsdk.Tool{
		Name:        td.OperationID,
		Description: td.Description,
		InputSchema: td.InputSchema,
	}

	if td.OutputSchema != nil {
		t.OutputSchema = td.OutputSchema
	}

	mcpsdk.AddTool(server, t, c.toolHandler(td))
}

func (c *Controller) toolHandler(td *models.ToolDefinition) func(context.Context, *mcpsdk.CallToolRequest, map[string]any) (*mcpsdk.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in map[string]any) (*mcpsdk.CallToolResult, any, error) {
		start := time.Now()

		ctx, span := c.tracer.Start(ctx, td.OperationID)
		defer span.End()

		attrs := metric.WithAttributes(attribute.String("tool", td.OperationID))

		c.metrics.ToolCalls.Add(ctx, 1, attrs)

		defer func() {
			c.metrics.ToolDuration.Record(ctx, time.Since(start).Seconds(), attrs)
		}()

		c.logger.InfoContext(ctx, "new tool call",
			"tool", td.OperationID,
			"method", td.Method,
			"path", td.Path,
		)

		status, body, err := c.tool.Execute(ctx, td, in)
		if err != nil {
			c.logger.ErrorContext(ctx, "error executing tool",
				"tool", td.OperationID,
				"error", err,
			)

			return nil, nil, err
		}

		text := string(body)

		var data any

		if err := json.Unmarshal(body, &data); err == nil {
			if td.WrapOutput && status >= 200 && status < 300 {
				data = map[string]any{"result": data}
			}

			if c.enableTOON {
				if encoded, err := toon.Marshal(data); err == nil {
					text = string(encoded)
				}
			} else {
				if minified, err := json.Marshal(data); err == nil {
					text = string(minified)
				}
			}
		}

		res := &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: fmt.Sprintf("HTTP %d", status)},
				&mcpsdk.TextContent{Text: text},
			},
		}

		return res, nil, nil
	}
}
