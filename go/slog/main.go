// Copyright 2020 Yoshi Yamaguchi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"log/slog"

	"cloud.google.com/go/compute/metadata"
	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/muncus/cloud-logging-configurations/go/slog/gcploghandler"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var logger *slog.Logger

func initTracing() (*sdktrace.TracerProvider, error) {
	var exporter sdktrace.SpanExporter
	exporter, err := texporter.New(texporter.WithProjectID(os.Getenv("GOOGLE_CLOUD_PROJECT")))
	if err != nil {
		// Fall back to printing traces to stdout, if we don't have a cloud project.
		log.Printf("Failed to create Cloud Trace exporter (%v). Falling back to stdout.\n", err)
		exporter, _ = stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
	res, err := resource.New(context.Background(),
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithAttributes(semconv.ServiceNameKey.String("loghandler")))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tp)
	return tp, nil
}

func init() {
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	var err error
	if project == "" {
		// attempt to auto-detect project ID if not specified.
		project, err = metadata.ProjectID()
		if err != nil {
			log.Printf("Failed to determine projectid: %v", err)
		}
	}
	logger = slog.New(gcploghandler.New(os.Stdout, &gcploghandler.Options{
		ProjectID: project,
	}))
}

func LogHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("This is info level log.")
	logger.Warn("This is waning level log.")
	logger.Error("This is error level log.")
	logger.ErrorContext(r.Context(), "This is error level log with context!.", "error", fmt.Errorf("this is an error"))
	// Log levels specific to GCP:
	logger.Log(r.Context(), gcploghandler.LevelCritical.Level(), "This is a critical log.")
	logger.Log(r.Context(), gcploghandler.LevelEmergency.Level(), "This is an emergency log!")

	if t := r.Header.Get("traceparent"); t != "" {
		fmt.Fprintf(w, "Found traceparent header: %s\n", t)
	}
	if t := r.Header.Get("x-cloud-trace-context"); t != "" {
		fmt.Fprintf(w, "Found xctc header: %s\n", t)
	}

	span := trace.SpanFromContext(r.Context())
	fmt.Fprintf(w, "Otel trace/span: %s / %s\n", span.SpanContext().TraceID(), span.SpanContext().SpanID())
	span.AddEvent("An event happened!")
	if span.SpanContext().IsSampled() {
		fmt.Fprintf(w, "Link To Trace: http://console.cloud.google.com/traces/list?&tid=%s\n",
			span.SpanContext().TraceID())
	} else {
		fmt.Fprintf(w, "Trace not sampled.")
	}

}

func main() {
	tp, err := initTracing()
	if err != nil {
		log.Println(err)
	} else {
		defer tp.Shutdown(context.Background())
	}
	wrappedHandler := otelhttp.NewHandler(http.HandlerFunc(LogHandler), "LogHandler")
	http.Handle("/", wrappedHandler)
	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// Start HTTP server.
	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
