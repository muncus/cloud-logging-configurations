# `log/slog` Sample

This sample shows use of the `log/slog` package introduced in Go 1.21, that is compatible with Google Cloud Logging.

When used with OpenTelemetry (http://opentelemetry.io), Trace information is also added when the logging method includes a context (e.g. `slog.WarnContext`).