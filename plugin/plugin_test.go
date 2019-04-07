package plugin_test

import (
	"github.com/alexandre-normand/slackscot/actions"
	"github.com/alexandre-normand/slackscot/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDefaultNewPlugin(t *testing.T) {
	p := plugin.New("loopy").Build()

	require.NotNil(t, p)
	assert.Equal(t, "loopy", p.Name)
	assert.False(t, p.NamespaceCommands)
	assert.Empty(t, p.Commands)
	assert.Empty(t, p.HearActions)
	assert.Empty(t, p.ScheduledActions)
}

func TestPluginWithSingleCommand(t *testing.T) {
	p := plugin.New("loopy").
		WithCommand(actions.NewCommand().Build()).
		Build()

	require.NotNil(t, p)
	assert.Len(t, p.Commands, 1)
	assert.Empty(t, p.HearActions)
	assert.Empty(t, p.ScheduledActions)
}

func TestPluginWithManyCommands(t *testing.T) {
	p := plugin.New("loopy").
		WithCommand(actions.NewCommand().WithUsage("command1").Build()).
		WithCommand(actions.NewCommand().WithUsage("command2").Build()).
		Build()

	require.NotNil(t, p)
	require.Len(t, p.Commands, 2)
	assert.Equal(t, "command1", p.Commands[0].Usage)
	assert.Equal(t, "command2", p.Commands[1].Usage)
	assert.Empty(t, p.HearActions)
	assert.Empty(t, p.ScheduledActions)
}

func TestPluginWithCommandsAndHearActions(t *testing.T) {
	p := plugin.New("loopy").
		WithCommand(actions.NewCommand().WithUsage("command").Build()).
		WithHearAction(actions.NewCommand().WithUsage("listener").Build()).
		Build()

	require.NotNil(t, p)
	require.Len(t, p.Commands, 1)
	assert.Equal(t, "command", p.Commands[0].Usage)
	require.Len(t, p.HearActions, 1)
	assert.Equal(t, "listener", p.HearActions[0].Usage)
	assert.Empty(t, p.ScheduledActions)
}

func TestPluginWithScheduledActions(t *testing.T) {
	p := plugin.New("loopy").
		WithScheduledAction(actions.NewScheduledAction().WithDescription("Check service status").Build()).
		Build()

	require.NotNil(t, p)
	require.Len(t, p.ScheduledActions, 1)
	assert.Equal(t, "Check service status", p.ScheduledActions[0].Description)
}

func TestPluginWithCommandNamespacing(t *testing.T) {
	p := plugin.New("loopy").
		WithCommandNamespacing().
		Build()

	require.NotNil(t, p)
	assert.True(t, p.NamespaceCommands)
}
