package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	traceSDK "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func exportSamplesAsTraces(samples []sample) {
	exporter, _ := stdouttrace.New(
		stdouttrace.WithWriter(os.Stderr),
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithoutTimestamps(),
	)

	tp := traceSDK.NewTracerProvider(
		traceSDK.WithBatcher(exporter),
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
				_, span := tracer.Start(ctx, p.p.Command, trace.WithTimestamp(p.startedAt))
				span.End(trace.WithTimestamp(sample.At))
				delete(procs, p.p.Pid)
			}
		}
	}
}
