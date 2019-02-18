// Package assertaction provides testing functions for validation a plugin action's behavior
package assertaction

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/stretchr/testify/assert"
	"testing"
)

// AnswerValidator is a function to do further validation of an action's answer. The return value is meant to be true if validation
// is successful and false otherwise (following the testify convention)
type AnswerValidator func(t *testing.T, a *slackscot.Answer) bool

// MatchesAndAnswers asserts that the action.Match is true and gets the action's answer to be further validated by AnswerValidator
func MatchesAndAnswers(t *testing.T, action slackscot.ActionDefinition, m *slackscot.IncomingMessage, validateAnswer AnswerValidator) bool {
	isMatch := action.Match(m)

	if assert.Equalf(t, true, isMatch, "Message [%s] expected to match but action.Match returned false", m.NormalizedText) {
		a := action.Answer(m)

		return validateAnswer(t, a)
	} else {
		return false
	}
}

// NotMatch asserts that action.Match is false
func NotMatch(t *testing.T, action slackscot.ActionDefinition, m *slackscot.IncomingMessage) bool {
	isMatch := action.Match(m)

	return assert.Equalf(t, false, isMatch, "Message [%s] should not be a match but action.Match returned true", m.NormalizedText)
}
