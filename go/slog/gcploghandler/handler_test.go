package gcploghandler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestSeverityAttr(t *testing.T) {
	tests := []struct {
		name  string
		level LogSeverity
		want  string
	}{
		{
			name:  "default",
			level: LevelDefault,
			want:  "DEFAULT",
		},
		{
			name:  "debug",
			level: LevelDebug,
			want:  "DEFAULT",
		},
		{
			name:  "info",
			level: LevelInfo,
			want:  "INFO",
		},
		{
			name:  "notice",
			level: LevelNotice,
			want:  "NOTICE",
		},
		{
			name:  "error",
			level: LevelError,
			want:  "ERROR",
		},
		{
			name:  "critical",
			level: LevelCritical,
			want:  "CRITICAL",
		},
		{
			name:  "alert",
			level: LevelAlert,
			want:  "ALERT",
		},
		{
			name:  "emergency",
			level: LevelEmergency,
			want:  "EMERGENCY",
		},
	}

	buf := new(bytes.Buffer)
	l := slog.New(New(buf, &Options{
		HandlerOptions: slog.HandlerOptions{
			Level: LevelDefault.Level(),
		},
		ProjectID: "",
	}))
	for _, tt := range tests {
		l.Log(context.TODO(), tt.level.Level(), "my message")
		if !strings.Contains(buf.String(), tt.want) {
			t.Errorf("TestSeverityAttr(%s): wanted to contain %s, got %s\n", tt.name, tt.want, buf.String())
		}

	}

}

type JSONLogEntry struct {
	Severity     string
	Message      string
	TraceID      string `json:"logging.googleapis.com/trace"`
	SpanID       string `json:"logging.googleapis.com/spanId"`
	TraceSampled bool   `json:"logging.googleapis.com/trace_sampled"`
}

func TestAttrRewrite(t *testing.T) {
	tests := []struct {
		name     string
		logargs  []any
		want     JSONLogEntry
		rewriter func([]string, slog.Attr) slog.Attr
	}{
		{name: "simple",
			logargs: []any{},
			want: JSONLogEntry{
				Severity: "INFO",
				Message:  "my-message",
			},
		},
		{name: "modify",
			logargs: []any{},
			want: JSONLogEntry{
				Severity: "INFO+rewritten",
				Message:  "my-message",
			},
			rewriter: func(s []string, a slog.Attr) slog.Attr {
				if a.Value.String() == "INFO" {
					a.Value = slog.StringValue("INFO+rewritten")
				}
				return a
			},
		},
		{name: "override",
			logargs: []any{},
			want: JSONLogEntry{
				Severity: "INFO",
				Message:  "", // changing the message key will fail to populate this field.
			},
			rewriter: func(s []string, a slog.Attr) slog.Attr {
				if a.Key == "message" {
					a.Key = slog.MessageKey // override back to default message key.
				}
				return a
			},
		},
	}

	for _, tt := range tests {
		buf := new(bytes.Buffer)
		l := slog.New(New(buf, &Options{
			HandlerOptions: slog.HandlerOptions{
				ReplaceAttr: tt.rewriter,
			},
			ProjectID: "",
		}))
		l.Log(context.Background(), LevelInfo.Level(), "my-message", tt.logargs...)
		entry := &JSONLogEntry{}
		err := json.Unmarshal(buf.Bytes(), entry)
		if err != nil {
			t.Fatalf("failed to parse json output: %v", err)
		}
		if *entry != tt.want {
			t.Errorf("unexpected diff: want %v, got %v", tt.want, entry)
		}
	}

}

func TestOtelAttrs(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{2},
		TraceFlags: 0,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	buf := new(bytes.Buffer)
	l := slog.New(New(buf, &Options{
		ProjectID: "my-project",
	}))
	l.WarnContext(ctx, "a warning")
	// parse log entry and check values.
	entry := &JSONLogEntry{}
	err := json.Unmarshal(buf.Bytes(), entry)
	if err != nil {
		t.Fatalf("failed to parse json output: %v", err)
	}
	if entry.TraceID != "projects/my-project/traces/01000000000000000000000000000000" {
		t.Errorf("TestOtelAttrs: unexpected traceid found: %s", entry.TraceID)
	}
	want_span := sc.SpanID().String()
	if entry.SpanID != want_span {
		t.Errorf("TestOtelAttrs: unexpected spanId: want %s got %s", want_span, entry.SpanID)
	}
}

func TestOtelWithoutProjectId(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{2},
		TraceFlags: 0,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	buf := &bytes.Buffer{}
	l := slog.New(New(buf, &Options{}))
	l.WarnContext(ctx, "a warning")

	entry := &JSONLogEntry{}
	err := json.Unmarshal(buf.Bytes(), entry)
	if err != nil {
		t.Fatalf("failed to parse json output: %v", err)
	}
	if entry.TraceID != "" {
		t.Errorf("TestOtelWithoutProjectId: want empty TraceID, got %s", entry.TraceID)
	}
	if entry.SpanID != "" {
		t.Errorf("TestOtelWithoutProjectId: want empty SpanID, got %s", entry.SpanID)
	}
}
