package observability

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig конфигурация для tracing
type TracingConfig struct {
	ServiceName    string `json:"service_name"    default:"family-budget-service"`
	ServiceVersion string `json:"service_version" default:"1.0.0"`
	OTLPEndpoint   string `json:"otlp_endpoint"   default:"http://localhost:4318"`
	Environment    string `json:"environment"     default:"development"`
	Enabled        bool   `json:"enabled"         default:"true"`
}

// InitTracing инициализирует OpenTelemetry tracing
func InitTracing(ctx context.Context, config TracingConfig, logger *slog.Logger) (func(context.Context) error, error) {
	if !config.Enabled {
		logger.InfoContext(ctx, "Tracing disabled")
		return func(context.Context) error { return nil }, nil
	}

	logger.InfoContext(ctx, "Initializing tracing",
		slog.String("otlp_endpoint", config.OTLPEndpoint),
		slog.String("service_name", config.ServiceName),
	)

	// Создаем OTLP HTTP exporter
	// Используем WithEndpointURL для полного контроля над URL
	fullURL := config.OTLPEndpoint + "/v1/traces"
	logger.InfoContext(ctx, "Creating OTLP exporter",
		slog.String("full_url", fullURL),
	)

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(fullURL),
		otlptracehttp.WithInsecure(), // для локальной разработки
	)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to create OTLP exporter", slog.String("error", err.Error()))
		return nil, err
	}

	// Создаем resource
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(config.ServiceName),
		semconv.ServiceVersionKey.String(config.ServiceVersion),
		semconv.DeploymentEnvironmentKey.String(config.Environment),
	)

	// Создаем trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Устанавливаем глобальный trace provider
	otel.SetTracerProvider(tp)

	// Устанавливаем глобальный propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.InfoContext(ctx, "Tracing initialized successfully",
		slog.String("service", config.ServiceName),
		slog.String("version", config.ServiceVersion),
		slog.String("otlp_endpoint", config.OTLPEndpoint),
	)

	// Возвращаем функцию для graceful shutdown
	return tp.Shutdown, nil
}

// Tracer структура для инкапсуляции трейсера
type Tracer struct {
	tracer trace.Tracer
}

// NewTracer создает новый экземпляр трейсера
func NewTracer(serviceName string) *Tracer {
	return &Tracer{
		tracer: otel.Tracer(serviceName),
	}
}

// StartSpan создает новый span с контекстом и возвращает функцию cleanup
// Возвращает функцию cleanup, которую следует вызвать (обычно через defer) для завершения span
func (t *Tracer) StartSpan(
	ctx context.Context,
	name string,
	opts ...trace.SpanStartOption,
) (context.Context, func()) {
	ctx, span := t.tracer.Start(ctx, name, opts...)
	cleanup := func() { span.End() }
	return ctx, cleanup
}

// StartSpanWithSpan создает новый span и возвращает сам span (для обратной совместимости)
// Caller должен вызвать span.End() для завершения span
func (t *Tracer) StartSpanWithSpan(
	ctx context.Context,
	name string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...) //nolint:spancheck // span.End() должен вызываться caller'ом
}

// GetTracer возвращает внутренний трейсер
func (t *Tracer) GetTracer() trace.Tracer {
	return t.tracer
}

// TraceRepository добавляет трейсинг к операциям с репозиторием
func (t *Tracer) TraceRepository(ctx context.Context, operation, collection string) (context.Context, trace.Span) {
	ctx, span := t.StartSpanWithSpan(ctx, "repository."+operation,
		trace.WithAttributes(
			attribute.String("db.operation", operation),
			attribute.String("db.collection.name", collection),
			attribute.String("db.system", "postgresql"),
		),
	)

	return ctx, span
}

// TraceHTTPRequest добавляет трейсинг к HTTP запросам
func (t *Tracer) TraceHTTPRequest(ctx context.Context, method, path string) (context.Context, trace.Span) {
	ctx, span := t.StartSpanWithSpan(ctx, "http."+method,
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.route", path),
		),
	)

	return ctx, span
}

// TraceBusiness добавляет трейсинг к бизнес-операциям
func (t *Tracer) TraceBusiness(
	ctx context.Context,
	domain, operation string,
	metadata map[string]string,
) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("business.domain", domain),
		attribute.String("business.operation", operation),
	}

	// Добавляем метаданные как атрибуты
	for key, value := range metadata {
		attrs = append(attrs, attribute.String("business."+key, value))
	}

	ctx, span := t.StartSpanWithSpan(ctx, "business."+domain+"."+operation,
		trace.WithAttributes(attrs...),
	)

	return ctx, span
}

// AddSpanAttributes добавляет атрибуты к текущему span
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

// AddSpanEvent добавляет событие к текущему span
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// RecordSpanError записывает ошибку в span
func RecordSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span != nil && err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("error", true))
	}
}

// SetSpanStatus устанавливает статус span
func SetSpanStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetStatus(code, description)
	}
}

// GetTraceID возвращает trace ID из контекста
func GetTraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// GetSpanID возвращает span ID из контекста
func GetSpanID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		return spanCtx.SpanID().String()
	}
	return ""
}
