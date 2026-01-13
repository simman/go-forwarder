package logger

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger initializes the global logger based on configuration
func InitLogger(level, format, output string) error {
	// Set log level
	logLevel, err := parseLevel(level)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(logLevel)

	// Set output writer
	var writer io.Writer
	switch output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		// Assume it's a file path
		f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		writer = f
	}

	// Set format
	if format == "text" {
		writer = zerolog.ConsoleWriter{Out: writer}
	}

	log.Logger = zerolog.New(writer).With().Timestamp().Caller().Logger()

	return nil
}

// parseLevel converts string level to zerolog.Level
func parseLevel(level string) (zerolog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	default:
		return zerolog.InfoLevel, nil
	}
}
