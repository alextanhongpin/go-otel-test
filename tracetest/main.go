package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/davecgh/go-spew/spew"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var exporter *tracetest.InMemoryExporter

func init() {
	// This does not work actually.
	// The expectation is that we can get the spans.
	exporter = tracetest.NewInMemoryExporter()
}

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

	span.SetAttributes(
		attribute.Int64("x", x),
		attribute.Int64("y", y),
	)
	span.SetStatus(codes.Ok, "success")
	span.AddEvent("hello")

	return x + y
}

func multiply(ctx context.Context, x, y int64) int64 {
	var span trace.Span
	_, span = tracer.Start(ctx, "Multiplication")
	defer span.End()

	span.RecordError(errors.New("bad multiplication"))

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
	tracerProvider := sdktrace.NewTracerProvider(
		// Change to Syncer because Batcher doesn't work.
		sdktrace.WithSyncer(exporter),
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
		// NOTE: Spans must be retrieved before shutting down, because Shutdown clears
		// the in-memory spans.

		spans := exporter.GetSpans().Snapshots()
		log.Println(len(spans))

		scs := &spew.ConfigState{
			DisablePointerAddresses: true,
			DisableCapacities:       true,
			Indent:                  "  ",
		}
		for _, span := range spans {
			scs.Dump(span)

			for _, attr := range span.Attributes() {
				log.Println(attr.Key, attr.Value.Emit())
			}
		}

		b, err := json.MarshalIndent(exporter.GetSpans().Snapshots(), "", " ")
		if err != nil {
			panic(err)
		}
		log.Println(string(b))
		// This will output readonly span.
		// 2023/07/01 21:53:30 [{"ReadOnlySpan":null},{"ReadOnlySpan":null},{"ReadOnlySpan":null}]

		b, err = json.MarshalIndent(exporter.GetSpans(), "", " ")
		if err != nil {
			panic(err)
		}

		// This will serialize the spans correctly.
		log.Println(string(b))

		if err := shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("the answer is", add(ctx, multiply(ctx, multiply(ctx, 2, 2), 10), 2))
}
