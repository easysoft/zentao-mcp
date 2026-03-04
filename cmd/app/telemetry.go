package main

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
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

	// Trace provider (only with OTLP endpoint).
	if endpoint != "" {
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
	}

	// Log provider: OTLP when endpoint is set, stdout otherwise.
	var logExp log.Exporter

	if endpoint != "" {
		opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlploggrpc.WithInsecure())
		}

		logExp, err = otlploggrpc.New(ctx, opts...)
	} else {
		logExp, err = stdoutlog.New()
	}

	if err != nil {
		return nil, fmt.Errorf("create log exporter: %w", err)
	}

	var logProcessor log.Processor
	if endpoint != "" {
		logProcessor = log.NewBatchProcessor(logExp)
	} else {
		logProcessor = log.NewSimpleProcessor(logExp)
	}

	lp := log.NewLoggerProvider(log.WithResource(res), log.WithProcessor(logProcessor))
	shutdowns = append(shutdowns, lp.Shutdown)

	// Global slog via otelslog bridge.
	handler := otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(lp))
	slog.SetDefault(slog.New(handler))

	return func(ctx context.Context) error {
		for _, fn := range shutdowns {
			if err := fn(ctx); err != nil {
				return err
			}
		}

		return nil
	}, nil
}
