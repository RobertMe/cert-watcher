// +build windows

package main

import (
	"github.com/rs/zerolog/log"
	"io"
)

func createJournaldLogWriter() io.Writer {
	log.Error().Str("location", "journald").Msg("Log location journald is not supported on Windows")
	return nil
}
