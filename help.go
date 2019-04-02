package slackscot

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/config"
	"io"
	"strings"
)

type helpPlugin struct {
	Plugin

	name                   string
	slackscotVersion       string
	timeLocation           string
	commands               map[string][]ActionDefinition
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

func (s *Slackscot) newHelpPlugin(version string) *helpPlugin {
	commands, hearActions, scheduledActions := findAllActions(s.namespaceCommands, s.plugins)

	helpPlugin := new(helpPlugin)
	helpPlugin.timeLocation = s.config.GetString(config.TimeLocationKey)
	helpPlugin.name = s.name
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

	fmt.Fprintf(&b, "I'm `%s` (engine `v%s`). I listen to the team's chat and provides automated functions :genie:.\n", h.name, h.slackscotVersion)

	if len(h.commands) > 0 {
		fmt.Fprintf(&b, "\nI currently support the following commands:\n")

		for n, commands := range h.commands {
			appendActions(&b, n, commands)
		}
	}

	if len(h.hearActions) > 0 {
		fmt.Fprintf(&b, "\nAnd listen for the following:\n")

		appendActions(&b, "", h.hearActions)
	}

	if len(h.pluginScheduledActions) > 0 {
		fmt.Fprintf(&b, "\nAnd do those things periodically:\n")

		appendScheduledActions(&b, h.timeLocation, h.pluginScheduledActions)
	}

	return &Answer{Text: b.String(), Options: []AnswerOption{AnswerInThread()}}
}

func appendActions(w io.Writer, pluginNamespace string, actions []ActionDefinition) {
	for _, value := range actions {
		if value.Usage != "" && !value.Hidden {
			if len(pluginNamespace) > 0 {
				fmt.Fprintf(w, "\t• `%s %s` - %s\n", pluginNamespace, value.Usage, value.Description)
			} else {
				fmt.Fprintf(w, "\t• `%s` - %s\n", value.Usage, value.Description)
			}
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

func findAllActions(namespaceCommands bool, plugins []*Plugin) (commands map[string][]ActionDefinition, hearActions []ActionDefinition, pluginScheduledActions []pluginScheduledAction) {
	commands = make(map[string][]ActionDefinition)
	hearActions = make([]ActionDefinition, 0)
	pluginScheduledActions = make([]pluginScheduledAction, 0)

	for _, p := range plugins {
		namespace := ""
		if namespaceCommands && p.NamespaceCommands {
			namespace = p.Name
		}

		if _, ok := commands[namespace]; !ok {
			commands[namespace] = make([]ActionDefinition, 0)
		}

		commands[namespace] = append(commands[namespace], p.Commands...)
		hearActions = append(hearActions, p.HearActions...)

		if p.ScheduledActions != nil {
			for _, sa := range p.ScheduledActions {
				pluginScheduledActions = append(pluginScheduledActions, pluginScheduledAction{plugin: p.Name, ScheduledActionDefinition: sa})
			}
		}
	}

	return commands, hearActions, pluginScheduledActions
}
