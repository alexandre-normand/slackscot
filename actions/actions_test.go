package actions_test

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/actions"
	"github.com/alexandre-normand/slackscot/schedule"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewCommandWithDefaults(t *testing.T) {
	action := actions.NewCommand().Build()
	assert.False(t, action.Hidden)
	assert.True(t, action.Match(&slackscot.IncomingMessage{}))
	assert.Nil(t, action.Answer(&slackscot.IncomingMessage{}))
}

func TestNewHearActionWithDefaults(t *testing.T) {
	action := actions.NewHearAction().Build()
	assert.False(t, action.Hidden)
	assert.True(t, action.Match(&slackscot.IncomingMessage{}))
	assert.Nil(t, action.Answer(&slackscot.IncomingMessage{}))
}

func TestNewActionWithMatcher(t *testing.T) {
	action := actions.NewHearAction().
		WithMatcher(func(m *slackscot.IncomingMessage) bool {
			return false
		}).
		Build()

	assert.False(t, action.Match(&slackscot.IncomingMessage{}))
}

func TestNewActionWithAnswerer(t *testing.T) {
	action := actions.NewHearAction().
		WithAnswerer(func(m *slackscot.IncomingMessage) *slackscot.Answer {
			return &slackscot.Answer{Text: "fake answer"}
		}).
		Build()

	assert.Equal(t, &slackscot.Answer{Text: "fake answer"}, action.Answer(&slackscot.IncomingMessage{}))
}

func TestNewActionWithUsage(t *testing.T) {
	action := actions.NewHearAction().
		WithUsage("make something").
		Build()

	assert.Equal(t, "make something", action.Usage)
}

func TestNewActionWithDescription(t *testing.T) {
	action := actions.NewHearAction().
		WithDescription("Instruct me to make something").
		Build()

	assert.Equal(t, "Instruct me to make something", action.Description)
}

func TestNewActionWithDescriptionf(t *testing.T) {
	action := actions.NewHearAction().
		WithDescriptionf("Instruct me to make one of %s", []string{"coffee", "soup"}).
		Build()

	assert.Equal(t, "Instruct me to make one of [coffee soup]", action.Description)
}

func TestNewHiddenAction(t *testing.T) {
	action := actions.NewHearAction().
		Hidden().
		Build()

	assert.True(t, action.Hidden)
}

func TestNewScheduledActionWithDefaults(t *testing.T) {
	action := actions.NewScheduledAction().Build()

	assert.False(t, action.Hidden)
	assert.Equal(t, schedule.Definition{}, action.Schedule)
	assert.NotPanics(t, assert.PanicTestFunc(action.Action))
}

func TestNewScheduledActionWithSchedule(t *testing.T) {
	action := actions.NewScheduledAction().WithSchedule(schedule.New().WithInterval(1, schedule.Hours).Build()).Build()

	assert.Equal(t, schedule.Definition{Interval: 1, Unit: schedule.Hours}, action.Schedule)
}

func TestNewScheduledActionWithDescription(t *testing.T) {
	action := actions.NewScheduledAction().
		WithDescription("Make a surprise").
		Build()

	assert.Equal(t, "Make a surprise", action.Description)
}

func TestNewScheduledActionWithDescriptionf(t *testing.T) {
	action := actions.NewScheduledAction().
		WithDescriptionf("Make one of %s", []string{"coffee", "soup"}).
		Build()

	assert.Equal(t, "Make one of [coffee soup]", action.Description)
}

func TestNewScheduledActionWithAction(t *testing.T) {
	action := actions.NewScheduledAction().
		WithAction(func() {
			panic("just checking that it's me")
		}).
		Build()

	assert.PanicsWithValue(t, "just checking that it's me", assert.PanicTestFunc(action.Action))
}
