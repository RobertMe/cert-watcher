package main

import (
	"github.com/RobertMe/cert-watcher/pkg/config/static"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"strings"
)

func configureLogging(forceDebug bool, config *static.Log) {
	level := getLogLevel(forceDebug, config.Level)
	zerolog.SetGlobalLevel(level)

	if writer := createLogWriter(config); writer != nil {
		log.Logger = zerolog.New(writer).With().Timestamp().Logger()
	}
}

func getLogLevel(forceDebug bool, logLevel string) zerolog.Level {
	var level zerolog.Level
	if forceDebug {
		level = zerolog.DebugLevel
	} else {
		switch strings.ToUpper(logLevel) {
		case "TRACE":
			level = zerolog.TraceLevel
			break
		case "DEBUG":
			level = zerolog.DebugLevel
			break
		case "INFO":
			level = zerolog.InfoLevel
			break
		case "WARN":
			level = zerolog.WarnLevel
			break
		case "ERROR":
			level = zerolog.ErrorLevel
			break
		case "FATAL":
			level = zerolog.FatalLevel
			break
		case "PANIC":
			level = zerolog.PanicLevel
			break
		case "DISABLED":
			level = zerolog.Disabled
			break
		default:
			level = zerolog.ErrorLevel
		}
	}
	return level
}

func createLogWriter(config *static.Log) io.Writer {
	var writers []io.Writer

	for _, location := range config.Location {
		switch strings.ToLower(location) {
		case "stdout":
			writers = append(writers, os.Stdout)
			break
		case "stderr":
			writers = append(writers, os.Stderr)
			break
		case "journald":
			if writer := createJournaldLogWriter(); writer != nil {
				writers = append(writers, writer)
			}
			break
		default:
			if strings.Contains(location, "/") {
				info, err := os.Stat(location)

				if info != nil && info.IsDir() {
					log.Warn().Str("location", location).Msg("Log location is a directory, not a file")
					continue
				}

				writer, err := os.OpenFile(location, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
				if err != nil {
					log.Error().Err(err).Str("location", location).Msg("Unable to open or create log file")
				}

				writers = append(writers, writer)
			}
		}
	}

	switch len(writers) {
	case 0:
		return nil
	case 1:
		return writers[0]
	default:
		return zerolog.MultiLevelWriter(writers...)
	}
}
