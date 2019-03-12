// Package plugins provides a collection of example (and usable) plugins for instances
// of slackscot
package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"strings"
)

// Versioner holds the plugin data for the karma plugin
type Versioner struct {
	slackscot.Plugin
}

const (
	versionerPluginName = "versionner"
)

// NewVersionner creates a new instance of the versioner plugin
func NewVersionner(name string, version string) *Versioner {
	return &Versioner{Plugin: slackscot.Plugin{Name: versionerPluginName, Commands: []slackscot.ActionDefinition{{
		Match: func(m *slackscot.IncomingMessage) bool {
			return strings.HasPrefix(m.NormalizedText, "version")
		},
		Usage:       "version",
		Description: fmt.Sprintf("Reply with `%s`'s `version` number", name),
		Answer: func(m *slackscot.IncomingMessage) *slackscot.Answer {
			return &slackscot.Answer{Text: fmt.Sprintf("I'm `%s`, version `%s`", name, version)}
		}}}, HearActions: nil}}
}
