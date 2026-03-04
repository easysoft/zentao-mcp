package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// initTelemetry sets up global slog (via otelslog bridge) and, when an OTLP
// endpoint is configured, the global TracerProvider. Returns a shutdown function.
func initTelemetry(ctx context.Context, cfg *TelemetryConfig) (func(context.Context) error, error) {
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	var shutdowns []func(context.Context) error

	endpoint := cfg.Endpoint

	if endpoint != "" {
		// Trace provider (only with OTLP endpoint).
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}

		traceExp, tErr := otlptracegrpc.New(ctx, opts...)
		if tErr != nil {
			return nil, fmt.Errorf("create trace exporter: %w", tErr)
		}

		tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithBatcher(traceExp))
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))

		shutdowns = append(shutdowns, tp.Shutdown)

		// Meter provider (only with OTLP endpoint).
		metricOpts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(endpoint)}
		if cfg.Insecure {
			metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
		}

		metricExp, mErr := otlpmetricgrpc.New(ctx, metricOpts...)
		if mErr != nil {
			return nil, fmt.Errorf("create metric exporter: %w", mErr)
		}

		mp := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)))
		otel.SetMeterProvider(mp)

		shutdowns = append(shutdowns, mp.Shutdown)

		// Logger provider (only with OTLP endpoint).
		logOpts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(endpoint)}
		if cfg.Insecure {
			logOpts = append(logOpts, otlploggrpc.WithInsecure())
		}

		logExp, lErr := otlploggrpc.New(ctx, logOpts...)
		if lErr != nil {
			return nil, fmt.Errorf("create log exporter: %w", lErr)
		}

		lp := log.NewLoggerProvider(log.WithResource(res), log.WithProcessor(log.NewBatchProcessor(logExp)))
		shutdowns = append(shutdowns, lp.Shutdown)

		handler := otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(lp))
		slog.SetDefault(slog.New(handler))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	}

	return func(ctx context.Context) error {
		for _, fn := range shutdowns {
			if err := fn(ctx); err != nil {
				return err
			}
		}

		return nil
	}, nil
}
