package cmd

import (
	"io"
	"log/slog"

	"github.com/lmittmann/tint"
)

func setLogger(output io.Writer, debug bool) {
	logLevel := &slog.LevelVar{}
	opts := &tint.Options{
		Level:     logLevel,
		AddSource: false,
	}

	if debug {
		logLevel.Set(slog.LevelDebug)
	} else {
		logLevel.Set(slog.LevelInfo)
	}

	handler := tint.NewHandler(output, opts)
	logger := slog.New(handler)

	slog.SetDefault(logger)
}
