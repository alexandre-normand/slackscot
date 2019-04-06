package plugin

import (
	"github.com/alexandre-normand/slackscot"
)

// PluginBuilder holds a plugin to build
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
func (pb *PluginBuilder) ScheduledAction(scheduledAction slackscot.ScheduledActionDefinition) *PluginBuilder {
	pb.plugin.ScheduledActions = append(pb.plugin.ScheduledActions, scheduledAction)
	return pb
}

// Build returns the created Plugin instance
func (pb *PluginBuilder) Build() (p *slackscot.Plugin) {
	return pb.plugin
}
