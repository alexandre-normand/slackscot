package slackscot

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"regexp"
	"strings"
)

type helpPlugin struct {
	Plugin
}

const (
	helpPluginName = "help"
)

// pluginScheduledAction represents a plugin's scheduled action with the plugin name and the action's definition
type pluginScheduledAction struct {
	plugin string
	ScheduledActionDefinition
}

func newHelpPlugin(name string, version string, c *viper.Viper, plugins []*Plugin) *helpPlugin {
	commands, hearActions, scheduledActions := findAllActions(plugins)

	return &helpPlugin{Plugin{Name: helpPluginName, Commands: []ActionDefinition{generateHelpCommand(c, name, version, commands, hearActions, scheduledActions)}, HearActions: nil}}
}

// generateHelpCommand generates a command providing a list of all of the slackscot commands and hear actions.
// Note that ActionDefinitions with the flag Hidden set to true won't be included in the list
func generateHelpCommand(c *viper.Viper, slackscotName string, version string, commands []ActionDefinition, hearActions []ActionDefinition, pluginScheduledActions []pluginScheduledAction) ActionDefinition {
	return ActionDefinition{
		Regex:       regexp.MustCompile("(?i)help"),
		Usage:       helpPluginName,
		Description: "Reply with usage instructions",
		Answerer: func(m *slack.Msg) string {
			var b strings.Builder

			fmt.Fprintf(&b, "I'm `%s` (engine version `%s`) that listens to the team's chat and provides automated functions.\n", slackscotName, version)

			if len(commands) > 0 {
				fmt.Fprintf(&b, "\nI currently support the following commands:\n")

				for _, value := range commands {
					if value.Usage != "" && !value.Hidden {
						fmt.Fprintf(&b, "\t• `%s` - %s\n", value.Usage, value.Description)
					}
				}
			}

			if len(hearActions) > 0 {
				fmt.Fprintf(&b, "\nAnd listen for the following:\n")

				for _, value := range hearActions {
					if value.Usage != "" && !value.Hidden {
						fmt.Fprintf(&b, "\t• `%s` - %s\n", value.Usage, value.Description)
					}
				}
			}

			if len(pluginScheduledActions) > 0 {
				fmt.Fprintf(&b, "\nAnd do those things periodically:\n")

				for _, value := range pluginScheduledActions {
					if !value.ScheduledActionDefinition.Hidden {
						fmt.Fprintf(&b, "\t• [`%s`] `%s` (`%s`) - %s\n", value.plugin, value.ScheduledActionDefinition.ScheduleDefinition, c.GetString(config.TimeLocationKey), value.ScheduledActionDefinition.Description)
					}
				}
			}

			return b.String()
		},
	}
}

func findAllActions(plugins []*Plugin) (commands []ActionDefinition, hearActions []ActionDefinition, pluginScheduledActions []pluginScheduledAction) {
	commands = make([]ActionDefinition, 0)
	hearActions = make([]ActionDefinition, 0)
	pluginScheduledActions = make([]pluginScheduledAction, 0)

	for _, p := range plugins {
		commands = append(commands, p.Commands...)
		hearActions = append(hearActions, p.HearActions...)

		if p.ScheduledActions != nil {
			for _, sa := range p.ScheduledActions {
				pluginScheduledActions = append(pluginScheduledActions, pluginScheduledAction{plugin: p.Name, ScheduledActionDefinition: sa})
			}
		}
	}

	return commands, hearActions, pluginScheduledActions
}
