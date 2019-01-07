// Package plugins provides a collection of example (and usable) plugins for instances
// of slackscot
package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/nlopes/slack"
	"strings"
)

// Versioner holds the plugin data for the karma plugin
type Versioner struct {
	slackscot.Plugin
}

const (
	versionerPluginName = "versioner"
)

// NewVersioner creates a new instance of the versioner plugin
func NewVersioner(name string, version string) *Versioner {
	return &Versioner{Plugin: slackscot.Plugin{Name: versionerPluginName, Commands: []slackscot.ActionDefinition{{
		Match: func(t string, m *slack.Msg) bool {
			return strings.HasPrefix(t, "version")
		},
		Usage:       "version",
		Description: fmt.Sprintf("Reply with `%s`'s `version` number", name),
		Answer: func(m *slack.Msg) string {
			return fmt.Sprintf("I'm `%s`, version `%s`", name, version)
		}}}, HearActions: nil}}
}
