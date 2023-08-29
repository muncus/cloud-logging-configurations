package gcploghandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"
)

const TraceIdKey = "logging.googleapis.com/trace"
const SpanIdKey = "logging.googleapis.com/spanId"
const TraceSampledKey = "logging.googleapis.com/trace_sampled"

type Options struct {
	slog.HandlerOptions
	ProjectID string
}

func New(w io.Writer, opts *Options) *GCPLogHandler {
	if opts == nil {
		opts = &Options{}
	}
	// insert our ReplaceAttr method before any requested.
	replacer := opts.ReplaceAttr
	opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		a = rewriteAttrs(groups, a)
		if replacer != nil {
			return replacer(groups, a)
		}
		return a
	}
	h := &GCPLogHandler{
		out:         w,
		JSONHandler: *slog.NewJSONHandler(w, &opts.HandlerOptions),
		opts:        *opts,
	}
	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}
	return h
}

// GCPLogHandler is a JSONHandler that produces output suitable for GCP structured logging
// See https://cloud.google.com/logging/docs/structured-logging for details.
type GCPLogHandler struct {
	slog.JSONHandler
	opts Options
	out  io.Writer
}

// func (h GCPLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
// 	return level >= h.opts.Level.Level()
// }

func (h GCPLogHandler) Handle(ctx context.Context, r slog.Record) error {
	// If project ID was specified, we can fill in trace fields.
	span := trace.SpanContextFromContext(ctx)
	if h.opts.ProjectID != "" && span.HasTraceID() {
		r.Add(TraceIdKey, fmt.Sprintf("projects/%s/traces/%s", h.opts.ProjectID, span.TraceID().String()))
		r.Add(SpanIdKey, span.SpanID())
		r.Add(TraceSampledKey, span.IsSampled())
	}
	return h.JSONHandler.Handle(ctx, r)
}

func rewriteAttrs(groups []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.LevelKey:
		a.Key = "severity"
		vint := a.Value.Any().(slog.Level)
		a.Value = slog.StringValue(LogSeverity(vint).LogValue())

	case slog.MessageKey:
		a.Key = "message"
	case slog.TimeKey:
		a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
	}
	return a

}
