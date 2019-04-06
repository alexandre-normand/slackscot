package actions

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
)

type ActionBuilder struct {
	action slackscot.ActionDefinition
}

func newAction() (ab *ActionBuilder) {
	ab = new(ActionBuilder)
	ab.action = slackscot.ActionDefinition{Hidden: false}

	// Default to always match. This is acceptable since we can accomplish the same
	// behavior most of the time by returning nil in the Answerer instead. For most cases,
	// this is fine as checking for a match still requires the Answerer to extract info
	// from the same matching logic. A simple matcher can be useful when the matching/triggering
	// logic can be made entirely separate from the answer logic (i.e. probability matching)
	ab.action.Match = func(m *slackscot.IncomingMessage) bool {
		return true
	}

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

func (ab *ActionBuilder) WithMatcher(matcher slackscot.Matcher) *ActionBuilder {
	ab.action.Match = matcher
	return ab
}

func (ab *ActionBuilder) WithUsage(usage string) *ActionBuilder {
	ab.action.Usage = usage
	return ab
}

func (ab *ActionBuilder) WithDescription(description string) *ActionBuilder {
	ab.action.Description = description
	return ab
}

func (ab *ActionBuilder) WithDescriptionf(format string, a ...interface{}) *ActionBuilder {
	ab.action.Description = fmt.Sprintf(format, a...)
	return ab
}

func (ab *ActionBuilder) WithAnswerer(answerer slackscot.Answerer) *ActionBuilder {
	ab.action.Answer = answerer
	return ab
}

func (ab *ActionBuilder) Hidden() *ActionBuilder {
	ab.action.Hidden = true
	return ab
}

func (ab *ActionBuilder) Build() slackscot.ActionDefinition {
	return ab.action
}
