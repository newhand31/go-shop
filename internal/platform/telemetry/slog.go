package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// logHandler adds trace_id and span_id to every record whose context carries a span. It decorates another handler,
// the same shape as a middleware: take one, return one.
type logHandler struct{ inner slog.Handler }

// NewLogHandler wraps inner. Log with the *Context methods (InfoContext, not Info): a plain logger.Info has no context
// to read the span from, so it quietly logs without the id.
func NewLogHandler(inner slog.Handler) slog.Handler { return logHandler{inner: inner} }

func (h logHandler) Enabled(ctx context.Context, l slog.Level) bool { return h.inner.Enabled(ctx, l) }

func (h logHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	return h.inner.Handle(ctx, r)
}

// WithAttrs and WithGroup must return a wrapped handler too. Forget that and logger.With("order", 7) hands back the
// inner handler, which has no idea what a trace id is: everything logged through that logger loses it, silently.
func (h logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return logHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h logHandler) WithGroup(name string) slog.Handler {
	return logHandler{inner: h.inner.WithGroup(name)}
}
