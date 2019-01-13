package slackscot_test

import (
	"github.com/alexandre-normand/slackscot/v2"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"testing"
)

func TestLogWhenDebugEnabled(t *testing.T) {
	var b strings.Builder
	l := log.New(&b, "", 0)
	slog := slackscot.NewSLogger(l, true)

	slog.Debugf("Writing a log statement for my little %s\n", "red bird")
	o := b.String()

	assert.Equal(t, "Writing a log statement for my little red bird\n", o)
}

func TestLogWhenDebugDisabled(t *testing.T) {
	var b strings.Builder
	l := log.New(&b, "", 0)
	slog := slackscot.NewSLogger(l, false)

	slog.Debugf("Writing a log statement for my little %s\n", "red bird")
	o := b.String()

	// Nothing should have been logged
	assert.Equal(t, "", o)
}

func TestPrintfLogsWhenDebugDisabled(t *testing.T) {
	var b strings.Builder
	l := log.New(&b, "", 0)
	slog := slackscot.NewSLogger(l, false)

	slog.Printf("Writing a log statement for my little %s\n", "red bird")
	o := b.String()

	assert.Equal(t, "Writing a log statement for my little red bird\n", o)
}

func TestPrintfLogsWhenDebugEnabled(t *testing.T) {
	var b strings.Builder
	l := log.New(&b, "", 0)
	slog := slackscot.NewSLogger(l, true)

	slog.Printf("Writing a log statement for my little %s\n", "red bird")
	o := b.String()

	assert.Equal(t, "Writing a log statement for my little red bird\n", o)
}
