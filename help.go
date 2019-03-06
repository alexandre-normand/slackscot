package slackscot

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/spf13/viper"
	"io"
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
		Match: func(m *IncomingMessage) bool {
			return strings.HasPrefix(m.NormalizedText, "help")
		},
		Usage:       helpPluginName,
		Description: "Reply with usage instructions",
		Answer:      helpPlugin.showHelp,
	}}, HearActions: nil}

	return helpPlugin
}

// showHelp generates a message providing a list of all of the slackscot commands and hear actions.
// Note that ActionDefinitions with the flag Hidden set to true won't be included in the list
func (h *helpPlugin) showHelp(m *IncomingMessage) *Answer {
	var b strings.Builder

	// Get the user's first name using the botservices
	userID := m.User
	user, err := h.UserInfoFinder.GetUserInfo(userID)
	if err != nil {
		h.Logger.Debugf("Error getting user info for user id [%s] so skipping mentioning the name (it would be awkward): %v", userID, err)
	} else {
		fmt.Fprintf(&b, "🤝 You're `%s` and ", user.RealName)
	}

	fmt.Fprintf(&b, "I'm `%s` (engine `v%s`). I listen to the team's chat and provides automated functions 🧞‍♂️.\n", h.name, h.slackscotVersion)

	if len(h.commands) > 0 {
		fmt.Fprintf(&b, "\nI currently support the following commands:\n")

		appendActions(&b, h.commands)
	}

	if len(h.hearActions) > 0 {
		fmt.Fprintf(&b, "\nAnd listen for the following:\n")

		appendActions(&b, h.hearActions)
	}

	if len(h.pluginScheduledActions) > 0 {
		fmt.Fprintf(&b, "\nAnd do those things periodically:\n")

		appendScheduledActions(&b, h.v.GetString(config.TimeLocationKey), h.pluginScheduledActions)
	}

	return &Answer{Text: b.String(), Options: []AnswerOption{AnswerInThread()}}
}

func appendActions(w io.Writer, actions []ActionDefinition) {
	for _, value := range actions {
		if value.Usage != "" && !value.Hidden {
			fmt.Fprintf(w, "\t• `%s` - %s\n", value.Usage, value.Description)
		}
	}
}

func appendScheduledActions(w io.Writer, timeLocationName string, scheduledActions []pluginScheduledAction) {
	for _, value := range scheduledActions {
		if !value.ScheduledActionDefinition.Hidden {
			fmt.Fprintf(w, "\t• [`%s`] `%s` (`%s`) - %s\n", value.plugin, value.ScheduledActionDefinition.Schedule, timeLocationName, value.ScheduledActionDefinition.Description)
		}
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
