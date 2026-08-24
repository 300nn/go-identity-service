package telemetry

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/300nn/go-identity-service/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type ShutdownFunc func(ctx context.Context) error

func Init(
	ctx context.Context,
	appCfg config.AppConfig,
	telemetryCfg config.TelemetryConfig,
	logger *slog.Logger,
) (ShutdownFunc, error) {
	if !telemetryCfg.Enabled {
		logger.Info("telemetry disabled")
		return noopShutdown, nil
	}

	exporter, err := newTraceExporter(ctx, telemetryCfg)
	if err != nil {
		return nil, err
	}

	res, err := newResource(appCfg)
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(
			sdktrace.TraceIDRatioBased(telemetryCfg.SampleRatio),
		),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	logger.Info(
		"telemetry initialized",
		"exporter", telemetryCfg.Exporter,
		"sample_ratio", telemetryCfg.SampleRatio,
	)

	return tracerProvider.Shutdown, nil
}

func newTraceExporter(
	ctx context.Context,
	cfg config.TelemetryConfig,
) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case "stdout":
		exporter, err := stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
		if err != nil {
			return nil, fmt.Errorf("create stdout trace exporter: %w", err)
		}

		return exporter, nil

	case "otlp":
		options := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
		}

		if cfg.Insecure {
			options = append(options, otlptracegrpc.WithInsecure())
		}

		exporter, err := otlptracegrpc.New(ctx, options...)
		if err != nil {
			return nil, fmt.Errorf("create otlp trace exporter: %w", err)
		}

		return exporter, nil

	default:
		return nil, fmt.Errorf("unsupported telemetry exporter: %s", cfg.Exporter)
	}
}

func newResource(appCfg config.AppConfig) (*resource.Resource, error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"",
			attribute.String("service.name", appCfg.Name),
			attribute.String("service.version", appCfg.Version),
			attribute.String("deployment.environment", appCfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}

	return res, nil
}

func noopShutdown(context.Context) error {
	return nil
}
