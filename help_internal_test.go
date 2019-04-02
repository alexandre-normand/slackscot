package slackscot

import (
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func newPluginWithActionsOfAllTypes() (p *Plugin) {
	p = new(Plugin)
	p.Name = "thank"
	p.NamespaceCommands = true
	p.Commands = []ActionDefinition{{
		Match: func(m *IncomingMessage) bool {
			return strings.HasPrefix(m.NormalizedText, "@user")
		},
		Usage:       "<someone of something to thank>",
		Description: "Format a thank you note",
		Answer: func(m *IncomingMessage) *Answer {
			return nil
		}}}

	p.HearActions = []ActionDefinition{{
		Match: func(m *IncomingMessage) bool {
			return strings.Contains(m.NormalizedText, "chickadee")
		},
		Usage:       "say `chickadee` and hear a chirp",
		Description: "Chirp when hearing people talk about chickadees",
		Answer: func(m *IncomingMessage) *Answer {
			return nil
		}}}

	p.ScheduledActions = []ScheduledActionDefinition{{Schedule: schedule.Definition{Interval: 30, Unit: schedule.Seconds}, Description: "Sends a heartbeat every 30 seconds", Action: func() {}}}

	return p
}

func TestHelpWithNamespacingEnabled(t *testing.T) {
	s, err := New("robert", config.NewViperWithDefaults())
	s.RegisterPlugin(newPluginWithActionsOfAllTypes())

	require.NoError(t, err)

	help := s.newHelpPlugin("1.0.0")
	help.UserInfoFinder = &userInfoFinder{}

	cmd := help.Commands[0]
	assert.False(t, cmd.Match(&IncomingMessage{NormalizedText: " help"}))
	require.True(t, cmd.Match(&IncomingMessage{NormalizedText: "help"}))
	assert.True(t, cmd.Match(&IncomingMessage{NormalizedText: "help and something else"}))

	a := cmd.Answer(&IncomingMessage{NormalizedText: "help"})
	require.NotNil(t, a)

	assert.Equal(t, "🤝 You're `Daniel Quinn` and I'm `robert` (engine `v1.0.0`). I listen to the team's chat and provides automated functions :genie:.\n\n"+
		"I currently support the following commands:\n\t• `thank <someone of something to thank>` - Format a thank you note\n\nAnd listen for the following:\n"+
		"\t• `say `chickadee` and hear a chirp` - Chirp when hearing people talk about chickadees\n\nAnd do those things periodically:\n"+
		"\t• [`thank`] `Every 30 seconds` (`Local`) - Sends a heartbeat every 30 seconds\n", a.Text)
}

func TestHelpWithNamespacingDisabled(t *testing.T) {
	s, err := New("robert", config.NewViperWithDefaults(), OptionNoPluginNamespacing())
	s.RegisterPlugin(newPluginWithActionsOfAllTypes())

	require.NoError(t, err)

	help := s.newHelpPlugin("1.0.0")
	help.UserInfoFinder = &userInfoFinder{}

	cmd := help.Commands[0]
	a := cmd.Answer(&IncomingMessage{NormalizedText: "help"})
	require.NotNil(t, a)

	assert.Equal(t, "🤝 You're `Daniel Quinn` and I'm `robert` (engine `v1.0.0`). I listen to the team's chat and provides automated functions :genie:.\n\n"+
		"I currently support the following commands:\n\t• `<someone of something to thank>` - Format a thank you note\n\nAnd listen for the following:\n"+
		"\t• `say `chickadee` and hear a chirp` - Chirp when hearing people talk about chickadees\n\nAnd do those things periodically:\n"+
		"\t• [`thank`] `Every 30 seconds` (`Local`) - Sends a heartbeat every 30 seconds\n", a.Text)
}
