// Package plugins provides a collection of example (and usable) plugins for instances
// of slackscot
package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/nlopes/slack"
	"regexp"
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
		Regex:       regexp.MustCompile("(?i)version"),
		Usage:       "version",
		Description: "Reply with the name and version of this slackscot instance",
		Answerer: func(m *slack.Msg) string {
			return fmt.Sprintf("I'm `%s`, version `%s`", name, version)
		}}}, HearActions: nil}}
}
