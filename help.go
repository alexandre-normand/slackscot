package slackscot

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/v2/config"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"strings"
)

type helpPlugin struct {
	Plugin

	v                      *viper.Viper
	name                   string
	slackscotVersion       string
	commands               []ActionDefinition
	hearActions            []ActionDefinition
	pluginScheduledActions []pluginScheduledAction
}

const (
	helpPluginName = "help"
)

// pluginScheduledAction represents a plugin's scheduled action with the plugin name and the action's definition
type pluginScheduledAction struct {
	plugin string
	ScheduledActionDefinition
}

func newHelpPlugin(name string, version string, viper *viper.Viper, plugins []*Plugin) *helpPlugin {
	commands, hearActions, scheduledActions := findAllActions(plugins)

	helpPlugin := new(helpPlugin)
	helpPlugin.v = viper
	helpPlugin.name = name
	helpPlugin.slackscotVersion = version
	helpPlugin.commands = commands
	helpPlugin.hearActions = hearActions
	helpPlugin.pluginScheduledActions = scheduledActions

	helpPlugin.Plugin = Plugin{Name: helpPluginName, Commands: []ActionDefinition{{
		Match: func(t string, m *slack.Msg) bool {
			return strings.HasPrefix(t, "help")
		},
		Usage:       helpPluginName,
		Description: "Reply with usage instructions",
		Answer:      helpPlugin.showHelp,
	}}, HearActions: nil}

	return helpPlugin
}

// showHelp generates a message providing a list of all of the slackscot commands and hear actions.
// Note that ActionDefinitions with the flag Hidden set to true won't be included in the list
func (h *helpPlugin) showHelp(m *slack.Msg) string {
	var b strings.Builder

	// Get the user's first name using the botservices
	userId := m.User
	user, err := h.UserInfoFinder.GetUserInfo(userId)
	if err != nil {
		h.Logger.Debugf("Error getting user info for user id [%s] so skipping mentioning the name (it would be awkward): %v", userId, err)
	} else {
		fmt.Fprintf(&b, "🤝 You're `%s` and ", user.RealName)
	}

	fmt.Fprintf(&b, "I'm `%s` (engine `v%s`). I listen to the team's chat and provides automated functions 🧞‍♂️.\n", h.name, h.slackscotVersion)

	if len(h.commands) > 0 {
		fmt.Fprintf(&b, "\nI currently support the following commands:\n")

		for _, value := range h.commands {
			if value.Usage != "" && !value.Hidden {
				fmt.Fprintf(&b, "\t• `%s` - %s\n", value.Usage, value.Description)
			}
		}
	}

	if len(h.hearActions) > 0 {
		fmt.Fprintf(&b, "\nAnd listen for the following:\n")

		for _, value := range h.hearActions {
			if value.Usage != "" && !value.Hidden {
				fmt.Fprintf(&b, "\t• `%s` - %s\n", value.Usage, value.Description)
			}
		}
	}

	if len(h.pluginScheduledActions) > 0 {
		fmt.Fprintf(&b, "\nAnd do those things periodically:\n")

		for _, value := range h.pluginScheduledActions {
			if !value.ScheduledActionDefinition.Hidden {
				fmt.Fprintf(&b, "\t• [`%s`] `%s` (`%s`) - %s\n", value.plugin, value.ScheduledActionDefinition.ScheduleDefinition, h.v.GetString(config.TimeLocationKey), value.ScheduledActionDefinition.Description)
			}
		}
	}

	return b.String()
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
