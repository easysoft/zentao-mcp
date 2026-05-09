package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/merzzzl/openapi-mcp-server/internal/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Execute runs an API operation with the given parameters.
func (s *Service) Execute(ctx context.Context, td *models.ToolDefinition, in map[string]any) (int, []byte, error) {
	ctx, span := s.tracer.Start(ctx, "Execute")
	defer span.End()

	s.logger.InfoContext(ctx, "executing tool",
		"path", td.Path,
		"method", td.Method,
		"input", redactInput(in),
	)

	u, err := url.Parse(s.baseURL)
	if err != nil {
		return 0, nil, fmt.Errorf("parse base URL: %w", err)
	}

	templ := s.rePath.ReplaceAllStringFunc(td.Path, func(m string) string {
		name := strings.Trim(m, "{}")

		if v, ok := in[name]; ok && v != nil {
			return url.PathEscape(fmt.Sprint(v))
		}

		return m
	})

	u.Path = path.Join(u.Path, templ)

	q := u.Query()

	for _, p := range td.Params {
		if p.In != "query" {
			continue
		}

		if v, ok := in[p.Name]; ok && v != nil {
			q.Add(p.Name, fmt.Sprint(v))
		}
	}

	u.RawQuery = q.Encode()

	s.logger.InfoContext(ctx, "upstream request prepared",
		"path", td.Path,
		"method", td.Method,
		"upstream_url", redactURL(u),
		"upstream_query_params", redactQueryParams(q),
	)

	var bodyReader io.Reader

	if td.HasBody {
		if payload, ok := in["payload"]; ok && payload != nil {
			jb, err := json.Marshal(payload)
			if err != nil {
				return 0, nil, fmt.Errorf("marshal payload: %w", err)
			}

			bodyReader = bytes.NewReader(jb)
		} else {
			s.logger.WarnContext(ctx, "request body expected but not provided",
				"path", td.Path,
				"method", td.Method,
			)
		}
	}

	req, err := http.NewRequestWithContext(ctx, td.Method, u.String(), bodyReader)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}

	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for _, p := range td.Params {
		if p.In != "header" {
			continue
		}

		if v, ok := in[p.Name]; ok && v != nil {
			req.Header.Set(p.Name, fmt.Sprint(v))
		}
	}

	httpStart := time.Now()

	resp, err := s.proxy.Do(ctx, req)
	if err != nil {
		return 0, nil, fmt.Errorf("do request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	const maxResponseSize = 10 << 20

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return 0, nil, fmt.Errorf("read response body: %w", err)
	}

	httpAttrs := metric.WithAttributes(attribute.String("http.request.method", td.Method), attribute.Int("http.response.status_code", resp.StatusCode))

	s.metrics.HTTPRequests.Add(ctx, 1, httpAttrs)
	s.metrics.HTTPDuration.Record(ctx, time.Since(httpStart).Seconds(), httpAttrs)

	s.logger.InfoContext(ctx, "tool executed",
		"path", td.Path,
		"method", td.Method,
		"status", resp.StatusCode,
	)

	return resp.StatusCode, body, nil
}
