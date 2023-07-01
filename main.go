package main

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var b bytes.Buffer

const (
	instrumentationName    = "github.com/instrumentron"
	instrumentationVersion = "v0.1.0"
)

var (
	tracer = otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(instrumentationVersion),
		trace.WithSchemaURL(semconv.SchemaURL),
	)
)

func add(ctx context.Context, x, y int64) int64 {
	var span trace.Span
	_, span = tracer.Start(ctx, "Addition")
	defer span.End()

	return x + y
}

func multiply(ctx context.Context, x, y int64) int64 {
	var span trace.Span
	_, span = tracer.Start(ctx, "Multiplication")
	defer span.End()

	return x * y
}

func Resource() *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("stdout-example"),
		semconv.ServiceVersion("0.0.1"),
	)
}

func InstallExportPipeline(ctx context.Context) (func(context.Context) error, error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(&b),
	)
	if err != nil {
		return nil, fmt.Errorf("creating stdout exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(Resource()),
	)
	otel.SetTracerProvider(tracerProvider)

	return tracerProvider.Shutdown, nil
}

func main() {
	ctx := context.Background()

	// Registers a tracer Provider globally.
	shutdown, err := InstallExportPipeline(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatal(err)
		}

		// The spans will only be available here after the shutdown..
		log.Println("YES")
		log.Println(string(b.Bytes()))
	}()

	log.Println("the answer is", add(ctx, multiply(ctx, multiply(ctx, 2, 2), 10), 2))
	log.Println("NOPE")
	log.Println(string(b.Bytes()))
}
