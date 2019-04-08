/*
Package actions provides a fluent API for creating slackscot plugin actions. Typical usages
will also involve using the plugin fluent API from github.com/alexandre-normand/slackscot/plugin.

Plugin examples using this API can be found in github.com/alexandre-normand/slackscot/plugins but
a quick one could look like:

	import (
		"github.com/alexandre-normand/slackscot"
		"github.com/alexandre-normand/slackscot/plugin"
		"github.com/alexandre-normand/slackscot/actions"
	)

	func newPlugin() (p *slackscot.Plugin) {
		p = plugin.New("maker").
		    WithCommand(actions.NewCommand().
				WithMatcher(func(m *slackscot.IncomingMessage) bool {
					return strings.HasPrefix(m.NormalizedText, "make")
				}).
				WithUsage("make <something>").
				WithDescription("Make the `<something>` you need").
				WithAnswerer(func(m *slackscot.IncomingMessage) *slackscot.Answer {
					return &slackscot.Answer{Text: fmt.Sprintf(":white_check_mark: It's ready for you!")}
				}).
				Build()
			 ).
			WithHearAction(actions.NewHearAction().
				Hidden().
				WithMatcher(func(m *slackscot.IncomingMessage) bool {
					return strings.HasPrefix(m.NormalizedText, "chirp")
				}).
				WithAnswerer(func(m *slackscot.IncomingMessage) *slackscot.Answer {
					return &slackscot.Answer{Text: "Did I hear a bird?"}
				}).
				Build()
		     ).
			WithScheduledAction(actions.NewScheduledAction().
				WithSchedule(schedule.New().Every(time.Monday.string()).AtTime("10:00").Build()).
				WithDescription("Start the week off").
				WithAction(weeklyKickoff).
				Build()
		     ).
			Build()
		return p
	}
*/
package actions

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/schedule"
)

// ActionBuilder holds the action to build
type ActionBuilder struct {
	action slackscot.ActionDefinition
}

// ScheduledActionBuilder holds the scheduled action to build
type ScheduledActionBuilder struct {
	scheduledAction slackscot.ScheduledActionDefinition
}

var (
	// Default to always match. This is acceptable since we can accomplish the same
	// behavior most of the time by returning nil in the Answerer instead. For most cases,
	// this is fine as checking for a match still requires the Answerer to extract info
	// from the same matching logic. A simple matcher can be useful when the matching/triggering
	// logic can be made entirely separate from the answer logic (i.e. probability matching)
	defaultMatcher = func(m *slackscot.IncomingMessage) bool {
		return true
	}

	// Default to always return nil. This is not a default you want to use in most cases
	defaultAnswerer = func(m *slackscot.IncomingMessage) *slackscot.Answer {
		return nil
	}
)

// newAction creates a new action and returns the ActionBuilder to set various attributes
// of the action. When done with the setup, the caller is expected to call Build() to get
// the action
func newAction() (ab *ActionBuilder) {
	ab = new(ActionBuilder)
	ab.action = slackscot.ActionDefinition{Hidden: false}

	ab.action.Match = defaultMatcher
	ab.action.Answer = defaultAnswerer

	return ab
}

// NewCommand returns a new ActionBuilder to build a new command
func NewCommand() (ab *ActionBuilder) {
	return newAction()
}

// NewHearAction returns a new ActionBuilder to build a new hear action
func NewHearAction() (ab *ActionBuilder) {
	return newAction()
}

// WithMatcher sets the action's matcher function
func (ab *ActionBuilder) WithMatcher(matcher slackscot.Matcher) *ActionBuilder {
	ab.action.Match = matcher
	return ab
}

// WithUsage sets the action usage
func (ab *ActionBuilder) WithUsage(usage string) *ActionBuilder {
	ab.action.Usage = usage
	return ab
}

// WithDescription sets the action description
func (ab *ActionBuilder) WithDescription(description string) *ActionBuilder {
	ab.action.Description = description
	return ab
}

// WithDescriptionf sets the action description delegating format and arguments to fmt.Sprintf
func (ab *ActionBuilder) WithDescriptionf(format string, a ...interface{}) *ActionBuilder {
	ab.action.Description = fmt.Sprintf(format, a...)
	return ab
}

// WithAnswerer sets the action's answerer function
func (ab *ActionBuilder) WithAnswerer(answerer slackscot.Answerer) *ActionBuilder {
	ab.action.Answer = answerer
	return ab
}

// Hidden sets the action to hidden
func (ab *ActionBuilder) Hidden() *ActionBuilder {
	ab.action.Hidden = true
	return ab
}

// Build returns the ActionDefinition
func (ab *ActionBuilder) Build() slackscot.ActionDefinition {
	return ab.action
}

// NewScheduledAction returns a new ScheduledActionBuilder to build a new ScheduledActionDefinition
func NewScheduledAction() (sab *ScheduledActionBuilder) {
	sab = new(ScheduledActionBuilder)
	sab.scheduledAction = slackscot.ScheduledActionDefinition{Hidden: false}
	sab.scheduledAction.Action = func() {}

	return sab
}

// WithSchedule sets the schedule for the scheduled action
func (sab *ScheduledActionBuilder) WithSchedule(schedule schedule.Definition) *ScheduledActionBuilder {
	sab.scheduledAction.Schedule = schedule
	return sab
}

// WithDescription sets the scheduled action description
func (sab *ScheduledActionBuilder) WithDescription(desc string) *ScheduledActionBuilder {
	sab.scheduledAction.Description = desc
	return sab
}

// WithDescriptionf sets the scheduled action description delegating format and arguments to fmt.Sprintf
func (sab *ScheduledActionBuilder) WithDescriptionf(format string, a ...interface{}) *ScheduledActionBuilder {
	sab.scheduledAction.Description = fmt.Sprintf(format, a...)
	return sab
}

// WithAction sets the action function to run on schedule
func (sab *ScheduledActionBuilder) WithAction(action slackscot.ScheduledAction) *ScheduledActionBuilder {
	sab.scheduledAction.Action = action
	return sab
}

// Build returns the ScheduledActionDefinition
func (sab *ScheduledActionBuilder) Build() slackscot.ScheduledActionDefinition {
	return sab.scheduledAction
}
