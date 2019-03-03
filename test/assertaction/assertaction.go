// Package assertaction provides testing functions for validation a plugin action's behavior
package assertaction

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/test/capture"
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
	}

	return false
}

// MatchesAndEmojiReacts asserts that the action.Match is true and validates that expected emojis reactions are added to the message
func MatchesAndEmojiReacts(t *testing.T, plugin *slackscot.Plugin, action slackscot.ActionDefinition, m *slackscot.IncomingMessage, expectedEmojis ...string) bool {
	isMatch := action.Match(m)

	if assert.Equalf(t, true, isMatch, "Message [%s] expected to match but action.Match returned false", m.NormalizedText) {
		// Register a new emoji reactor with the plugin
		ec := capture.NewEmojiReactor()
		plugin.EmojiReactor = ec

		action.Answer(m)

		return assert.Equalf(t, m.Channel, ec.Channel, "Expected emoji reactions on the same channel as the triggering message [%s] but was [%s]", m.Channel, ec.Channel) &&
			assert.Equalf(t, m.Timestamp, ec.Timestamp, "Expected emoji reactions on the same timestamp as the triggering message [%s] but was [%s]", m.Timestamp, ec.Timestamp) &&
			assert.ElementsMatchf(t, expectedEmojis, ec.Emojis, "Expected emoji reactions [%s] but got [%s]", expectedEmojis, ec.Emojis)
	}

	return false
}

// NotMatch asserts that action.Match is false
func NotMatch(t *testing.T, action slackscot.ActionDefinition, m *slackscot.IncomingMessage) bool {
	isMatch := action.Match(m)

	return assert.Equalf(t, false, isMatch, "Message [%s] should not be a match but action.Match returned true", m.NormalizedText)
}
