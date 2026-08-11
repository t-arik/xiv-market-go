package main

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	logglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	logsdk "go.opentelemetry.io/otel/sdk/log"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.39.0"
)

const instrumentationName = "github.com/t-arik/xiv-market-go/cmd/export"

func setupTelemetry(ctx context.Context) (func(context.Context) error, error) {
	res, err := resource.New(
		ctx,
		resource.WithAttributes(semconv.ServiceName("xiv-market-export")),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, err
	}

	spans, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, err
	}

	metrics, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return nil, err
	}

	logs, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(spans),
		tracesdk.WithResource(res),
	)
	mp := metricsdk.NewMeterProvider(
		metricsdk.WithReader(metrics),
		metricsdk.WithResource(res),
	)
	lp := logsdk.NewLoggerProvider(
		logsdk.WithProcessor(logsdk.NewBatchProcessor(logs)),
		logsdk.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	logglobal.SetLoggerProvider(lp)
	slog.SetDefault(otelslog.NewLogger(instrumentationName, otelslog.WithLoggerProvider(lp)))

	return func(ctx context.Context) error {
		return errors.Join(
			lp.Shutdown(ctx),
			mp.Shutdown(ctx),
			tp.Shutdown(ctx),
		)
	}, nil
}
