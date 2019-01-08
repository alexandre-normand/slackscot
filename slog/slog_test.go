package slog

import (
	"github.com/alexandre-normand/slackscot/v2/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"testing"
)

func TestLogWhenDebugEnabled(t *testing.T) {
	viper.Set(config.DebugKey, true)

	var b strings.Builder
	l := log.New(&b, "", 0)
	Debugf(l, "Writing a log statement for my little %s\n", "red bird")

	o := b.String()

	assert.Equal(t, "Writing a log statement for my little red bird\n", o)
}

func TestLogWhenDebugDisabled(t *testing.T) {
	viper.Set(config.DebugKey, false)

	var b strings.Builder
	l := log.New(&b, "", 0)
	Debugf(l, "Writing a log statement for my little %s\n", "red bird")

	o := b.String()

	// Nothing should have been logged
	assert.Equal(t, "", o)
}
