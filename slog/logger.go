package slog

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/spf13/viper"
	"log"
)

// Debugf logs a debug line after checking if the configuration is in debug mode
func Debugf(l *log.Logger, format string, v ...interface{}) {
	if viper.GetBool(config.DebugKey) {
		l.Output(3, fmt.Sprintf(format, v...))
	}
}
