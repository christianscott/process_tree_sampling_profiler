package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func exportSamplesAsTraces(samples []sample) {
	exporter, _ := stdouttrace.New(
		stdouttrace.WithWriter(os.Stderr),
		stdouttrace.WithPrettyPrint(),
		// stdouttrace.WithoutTimestamps(),
	)

	resource, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("fib"),
			semconv.ServiceVersionKey.String("v0.1.0"),
			attribute.String("environment", "demo"),
		),
	)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource),
	)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()
	otel.SetTracerProvider(tp)

	tracer := otel.Tracer(NAME)
	ctx, execSpan := tracer.Start(context.Background(), "start")
	defer execSpan.End()

	procs := make(map[int]struct {
		p         proc
		startedAt time.Time
	})
	for i, sample := range samples {
		for _, p := range sample.Procs {
			_, seenBefore := procs[p.Pid]
			if !seenBefore {
				// proc started
				procs[p.Pid] = struct {
					p         proc
					startedAt time.Time
				}{p, sample.At}
			}
		}
		for _, p := range procs {
			_, procStillRunning := sample.Procs[p.p.Pid]
			if !procStillRunning || i == len(samples)-1 {
				// proc ended, send span
				_, span := tracer.Start(ctx, p.p.Command)
				defer span.End()

				// does not actually overwrite the start and end time :(
				span.SetAttributes(attribute.Int64("StartTime", 0))
				span.SetAttributes(attribute.Int64("end", sample.At.Unix()))

				delete(procs, p.p.Pid)
			}
		}
	}
}
