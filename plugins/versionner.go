// Package plugins provides a collection of example (and usable) plugins for instances
// of slackscot
package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/actions"
	"github.com/alexandre-normand/slackscot/plugin"
	"strings"
)

const (
	versionnerPluginName = "versionner"
)

// NewVersionner creates a new instance of the versionner plugin
func NewVersionner(name string, version string) (p *slackscot.Plugin) {
	p = plugin.New(versionnerPluginName).
		WithCommand(actions.New().
			WithMatcher(func(m *slackscot.IncomingMessage) bool {
				return strings.HasPrefix(m.NormalizedText, "version")
			}).
			WithUsage("version").
			WithDescriptionf("Reply with `%s`'s `version` number", name).
			WithAnswerer(func(m *slackscot.IncomingMessage) *slackscot.Answer {
				return &slackscot.Answer{Text: fmt.Sprintf("I'm `%s`, version `%s`", name, version)}
			}).
			Build()).
		Build()
	return p
}
