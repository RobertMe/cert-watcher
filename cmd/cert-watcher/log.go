package main

import (
	"github.com/RobertMe/cert-watcher/pkg/config/static"
	"github.com/rs/zerolog"
	"strings"
)

func configureLogging(forceDebug bool, log *static.Log) {
	level := getLogLevel(forceDebug, log.Level)
	zerolog.SetGlobalLevel(level)
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
