/*
Package plugin provides a fluent API for creating slackscot plugins. Typical usages
will also involve using the actions fluent API from github.com/alexandre-normand/slackscot/actions.

Plugin examples using this API can be found in github.com/alexandre-normand/slackscot/plugins but
a quick one could look like:

	import (
		"github.com/alexandre-normand/slackscot"
		"github.com/alexandre-normand/slackscot/plugin"
		"github.com/alexandre-normand/slackscot/actions"
	)

	func newPlugin() (p *slackscot.Plugin) {
		p = plugin.New("maker").
		    WithCommand(actions.NewCommand().
				WithMatcher(func(m *slackscot.IncomingMessage) bool {
					return strings.HasPrefix(m.NormalizedText, "make")
				}).
				WithUsage("make <something>").
				WithDescription("Make the `<something>` you need").
				WithAnswerer(func(m *slackscot.IncomingMessage) *slackscot.Answer {
					return &slackscot.Answer{Text: fmt.Sprintf(":white_check_mark: It's ready for you!")}
				}).
				Build()).
			WithHearAction(actions.NewHearAction().
				Hidden().
				WithMatcher(func(m *slackscot.IncomingMessage) bool {
					return strings.HasPrefix(m.NormalizedText, "chirp")
				}).
				WithAnswerer(func(m *slackscot.IncomingMessage) *slackscot.Answer {
					return &slackscot.Answer{Text: "Did I hear a bird?"}
				}).
				Build()
		     ).
			Build()
		return p
	}
*/
package plugin

import (
	"github.com/alexandre-normand/slackscot"
)

// PluginBuilder holds a plugin to build. This is used to set up
// a plugin that can be returned using Build
type PluginBuilder struct {
	plugin *slackscot.Plugin
}

// New creates a new PluginBuilder with a plugin with the given name and empty set of actions
func New(name string) (pb *PluginBuilder) {
	pb = new(PluginBuilder)
	pb.plugin = new(slackscot.Plugin)
	pb.plugin.Name = name
	pb.plugin.Commands = make([]slackscot.ActionDefinition, 0)
	pb.plugin.HearActions = make([]slackscot.ActionDefinition, 0)
	pb.plugin.ScheduledActions = make([]slackscot.ScheduledActionDefinition, 0)

	return pb
}

// WithCommand adds a command to the plugin
func (pb *PluginBuilder) WithCommand(command slackscot.ActionDefinition) *PluginBuilder {
	pb.plugin.Commands = append(pb.plugin.Commands, command)
	return pb
}

// WithHearAction adds an hear action to the plugin
func (pb *PluginBuilder) WithHearAction(hearAction slackscot.ActionDefinition) *PluginBuilder {
	pb.plugin.HearActions = append(pb.plugin.HearActions, hearAction)
	return pb
}

// WithCommandNamespacing enables command namespacing for that plugin
func (pb *PluginBuilder) WithCommandNamespacing() *PluginBuilder {
	pb.plugin.NamespaceCommands = true
	return pb
}

// WithScheduledAction adds a scheduled action to the plugin
func (pb *PluginBuilder) WithScheduledAction(scheduledAction slackscot.ScheduledActionDefinition) *PluginBuilder {
	pb.plugin.ScheduledActions = append(pb.plugin.ScheduledActions, scheduledAction)
	return pb
}

// Build returns the created Plugin instance
func (pb *PluginBuilder) Build() (p *slackscot.Plugin) {
	return pb.plugin
}
