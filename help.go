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
	prefix                 string
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
	if !strings.HasPrefix(s.selfUserPrefix, "<@") {
		helpPlugin.prefix = s.selfUserPrefix
	}

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
		fmt.Fprintf(&b, "🤝 Hi, `%s`! ", user.RealName)
	}

	fmt.Fprintf(&b, "I'm `%s` (engine `v%s`) and I listen to the team's chat and provides automated functions :genie:.\n", h.name, h.slackscotVersion)

	if lenCommands(h.commands) > 0 {
		fmt.Fprintf(&b, "\nI currently support the following commands:\n")

		for n, commands := range h.commands {
			appendActions(&b, h.prefix, n, commands)
		}
	}

	if len(h.hearActions) > 0 {
		fmt.Fprintf(&b, "\nAnd listen for the following:\n")

		appendActions(&b, "", "", h.hearActions)
	}

	if len(h.pluginScheduledActions) > 0 {
		fmt.Fprintf(&b, "\nAnd do those things periodically:\n")

		appendScheduledActions(&b, h.timeLocation, h.pluginScheduledActions)
	}

	return &Answer{Text: b.String(), Options: []AnswerOption{AnswerInThread()}}
}

// lenCommands returns the length of a map of string to array of values by summing
// up the length of all array values
func lenCommands(entries map[string][]ActionDefinition) (length int) {
	length = 0
	for _, v := range entries {
		length = length + len(v)
	}

	return length
}

func appendActions(w io.Writer, prefix string, pluginNamespace string, actions []ActionDefinition) {
	for _, value := range actions {
		if value.Usage != "" && !value.Hidden {
			if len(pluginNamespace) > 0 {
				fmt.Fprintf(w, "\t• `%s%s %s` - %s\n", prefix, pluginNamespace, value.Usage, value.Description)
			} else {
				fmt.Fprintf(w, "\t• `%s%s` - %s\n", prefix, value.Usage, value.Description)
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

		commands[namespace] = append(commands[namespace], filterNonHiddenActions(p.Commands)...)
		hearActions = append(hearActions, filterNonHiddenActions(p.HearActions)...)
		pluginScheduledActions = append(pluginScheduledActions, filterNonHiddenScheduledActions(p.Name, p.ScheduledActions)...)
	}

	return commands, hearActions, pluginScheduledActions
}

func filterNonHiddenActions(actions []ActionDefinition) (visibleActions []ActionDefinition) {
	visibleActions = make([]ActionDefinition, 0)
	for _, a := range actions {
		if !a.Hidden {
			visibleActions = append(visibleActions, a)
		}
	}

	return visibleActions
}

func filterNonHiddenScheduledActions(pluginName string, actions []ScheduledActionDefinition) (visibleActions []pluginScheduledAction) {
	visibleActions = make([]pluginScheduledAction, 0)

	for _, sa := range actions {
		if !sa.Hidden {
			visibleActions = append(visibleActions, pluginScheduledAction{plugin: pluginName, ScheduledActionDefinition: sa})
		}
	}

	return visibleActions
}
