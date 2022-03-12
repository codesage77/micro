package service

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func initTracer(name string, version string, sampler sdktrace.Sampler, exporter sdktrace.SpanExporter) (*sdktrace.TracerProvider, error) {
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(name))
	semconv.ServiceVersionKey.String(version)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
	)

	otel.SetTracerProvider(tp)

	return tp, nil
}

func trace(h http.Handler, tp *sdktrace.TracerProvider, tracerName string, spanName string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tp.Tracer(tracerName)
		ctx, span := tracer.Start(r.Context(), spanName)
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.target", r.RequestURI),
			attribute.String("http.host", r.Host),
			attribute.String("http.scheme", r.URL.Scheme),
			// attribute.Int("http.status", r.Response.StatusCode),
		)

		sr := r.WithContext(ctx)
		h.ServeHTTP(w, sr)
	})
}
