package slackscot

import (
	"fmt"
	"log"
)

// SLogger is the slackscot internal logging interface. The standard library logger implements this interface
type SLogger interface {
	Printf(format string, v ...interface{})

	Debugf(format string, v ...interface{})
}

type sLogger struct {
	logger *log.Logger
	debug  bool
}

// NewSLogger creates a new Slackscot logger provided with an interface logger and a debug flag
func NewSLogger(log *log.Logger, debug bool) (l *sLogger) {
	sl := new(sLogger)
	sl.debug = debug
	sl.logger = log
	return sl
}

// Debugf logs a debug line after checking if the configuration is in debug mode
func (sl *sLogger) Debugf(format string, v ...interface{}) {
	if sl.debug {
		sl.Printf(fmt.Sprintf(format, v...))
	}
}

// Printf logs a line by delegating the call to Output
func (sl *sLogger) Printf(format string, v ...interface{}) {
	sl.logger.Output(2, fmt.Sprintf(format, v...))
}
