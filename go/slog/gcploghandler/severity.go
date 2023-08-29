package gcploghandler

import "log/slog"

type LogSeverity int

func (s LogSeverity) LogValue() string {
	if v, ok := LevelNames[s]; ok {
		return v
	}
	// fall back to standard strings.
	return slog.Level(s).String()
}
func (s LogSeverity) Level() slog.Level {
	return slog.Level(s)
}

// Level names come from https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#logseverity
// Note that these numeric values *do not* match the above link. Only the string values from LevelNames are used in logs.
const (
	LevelDefault   = LogSeverity(-8)
	LevelDebug     = LogSeverity(slog.LevelDebug)
	LevelInfo      = LogSeverity(slog.LevelInfo)
	LevelNotice    = LogSeverity(2)
	LevelWarning   = LogSeverity(slog.LevelWarn)
	LevelError     = LogSeverity(slog.LevelError)
	LevelCritical  = LogSeverity(60)
	LevelAlert     = LogSeverity(70)
	LevelEmergency = LogSeverity(80)
)

var LevelNames = map[LogSeverity]string{
	LevelDefault:   "DEFAULT",
	LevelDebug:     "DEBUG",
	LevelInfo:      "INFO",
	LevelNotice:    "NOTICE",
	LevelWarning:   "WARNING",
	LevelError:     "ERROR",
	LevelCritical:  "CRITICAL",
	LevelAlert:     "ALERT",
	LevelEmergency: "EMERGENCY",
}
