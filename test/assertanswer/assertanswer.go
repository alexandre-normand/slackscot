// Package assertanswer provides testing functions to validate a plugin's answer
package assertanswer

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/stretchr/testify/assert"
	"testing"
)

// ResolvedAnswerOption holds a pair of Key/Value representing the physical AnswerOption
type ResolvedAnswerOption struct {
	Key   string
	Value string
}

// HasText asserts that the answer's text is the expected text
func HasText(t *testing.T, answer *slackscot.Answer, text string) bool {
	if assert.NotNil(t, answer) {
		return assert.Equalf(t, text, answer.Text, "Answer text expected to be [%s] but was [%s]", text, answer.Text)
	}
	return false
}

// HasText asserts that the answer's text contains the expected subString
func HasTextContaining(t *testing.T, answer *slackscot.Answer, subString string) bool {
	if assert.NotNil(t, answer) {
		return assert.Containsf(t, answer.Text, subString, "Answer expected to have text containing [%s] but its text [%s] didn't", subString, answer.Text)
	}
	return false
}

// HasOptions asserts that the answer's options contains the expected configuration key/values
func HasOptions(t *testing.T, answer *slackscot.Answer, options ...ResolvedAnswerOption) bool {
	if assert.NotNil(t, answer) {
		ropts := convertConfigsToResolvedAnswerOptions(slackscot.ApplyAnswerOpts(answer.Options...))
		return assert.ElementsMatchf(t, options, ropts, "Answer options expected %s but were %s", options, ropts)
	}
	return false
}

// convertConfigsToResolvedAnswerOptions converts a map[string]string of answer options to an array
// of ResolvedAnswerOptions for easier matching
func convertConfigsToResolvedAnswerOptions(configs map[string]string) (ropts []ResolvedAnswerOption) {
	ropts = make([]ResolvedAnswerOption, 0)

	for key, value := range configs {
		ropts = append(ropts, ResolvedAnswerOption{Key: key, Value: value})
	}

	return ropts
}
