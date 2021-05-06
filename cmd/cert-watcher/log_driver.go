// +build !windows

package main

import (
	"github.com/rs/zerolog/journald"
	"io"
)

func createJournaldLogWriter() io.Writer {
	return journald.NewJournalDWriter()
}
