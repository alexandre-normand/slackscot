package slackscot

import (
	"fmt"
	"github.com/nlopes/slack"
	"regexp"
)

type helpPlugin struct {
	Plugin
}

const (
	helpPluginName = "help"
)

func newHelpPlugin(name string, version string, plugins []*Plugin) *helpPlugin {
	commands, hearActions := findAllActions(plugins)

	return &helpPlugin{Plugin{Name: helpPluginName, Commands: []ActionDefinition{generateHelpCommand(name, version, commands, hearActions)}, HearActions: nil}}
}

// generateHelpCommand generates a command providing a list of all of the slackscot commands and hear actions.
// Note that ActionDefinitions with the flag Hidden set to true won't be included in the list
func generateHelpCommand(slackscotName string, version string, commands []ActionDefinition, hearActions []ActionDefinition) ActionDefinition {
	return ActionDefinition{
		Regex:       regexp.MustCompile("(?i)help"),
		Usage:       "help",
		Description: "Reply with usage instructions",
		Answerer: func(m *slack.Msg) string {
			response := fmt.Sprintf("I'm `%s` (engine version `%s`) that listens to the team's chat and provides automated functions."+
				"  I currently support the following commands:\n", slackscotName, version)

			for _, value := range commands {
				if value.Usage != "" && !value.Hidden {
					response = fmt.Sprintf("%s\n\t%s", response, value)
				}
			}

			for _, value := range hearActions {
				if value.Usage != "" && !value.Hidden {
					response = fmt.Sprintf("%s\n\t%s", response, value)
				}
			}

			return response
		},
	}
}

func findAllActions(plugins []*Plugin) (commands []ActionDefinition, hearActions []ActionDefinition) {
	commands = make([]ActionDefinition, 0)
	hearActions = make([]ActionDefinition, 0)

	for _, p := range plugins {
		commands = append(commands, p.Commands...)
		hearActions = append(hearActions, p.HearActions...)
	}

	return commands, hearActions
}
