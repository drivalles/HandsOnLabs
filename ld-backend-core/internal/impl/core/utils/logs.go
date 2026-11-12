package ldutils

import (
	"log/slog"
	"os"

	slogmulti "github.com/samber/slog-multi"
	"gopkg.in/natefinch/lumberjack.v2"
)

var HttpLogger *slog.Logger

func SetFileRotationLogs(appName string, env string, version string, dir string) {
	rotator := &lumberjack.Logger{
		Filename:   dir + "/app.log",
		MaxSize:    10,   // megabytes before rotation
		MaxBackups: 3,    // number of old log files to keep
		MaxAge:     28,   // days to retain old files
		Compress:   true, // whether to compress old logs (gzip)
	}

	logger := slog.New(
		slogmulti.Fanout(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}),
			slog.NewJSONHandler(rotator, &slog.HandlerOptions{}),
		),
	)
	logger = logger.With("app", appName, "env", env, "server_version", version)
	slog.SetDefault(logger)
	HttpLogger = setHttpLogger(appName, env, version, dir)
}

func setHttpLogger(appName string, env string, version string, dir string) *slog.Logger {
	rotator := &lumberjack.Logger{
		Filename:   dir + "/http.log",
		MaxSize:    10,   // megabytes before rotation
		MaxBackups: 3,    // number of old log files to keep
		MaxAge:     28,   // days to retain old files
		Compress:   true, // whether to compress old logs (gzip)
	}

	logger := slog.New(
		slogmulti.Fanout(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}),
			slog.NewJSONHandler(rotator, &slog.HandlerOptions{}),
		),
	)
	return logger.With("app", appName, "env", env, "server_version", version)
}
