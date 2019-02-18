package assertanswer_test

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/test/assertanswer"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHasTextNoMatch(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, false, assertanswer.HasText(mockT, &slackscot.Answer{Text: "this is my final answer"}, "this is my first answer"))
}

func TestHasTextNilAnswer(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, false, assertanswer.HasText(mockT, nil, "this is my first answer"))
}

func TestHasTextMatch(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, true, assertanswer.HasText(mockT, &slackscot.Answer{Text: "this is my final answer"}, "this is my final answer"))
}

func TestHasTextContainingMatch(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, true, assertanswer.HasTextContaining(mockT, &slackscot.Answer{Text: "this is my final answer"}, "final"))
}

func TestHasTextContainingNoMatch(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, false, assertanswer.HasTextContaining(mockT, &slackscot.Answer{Text: "this is my final answer"}, "the gopher always has more answers"))
}

func TestHasTextContainingNilAnswer(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, false, assertanswer.HasTextContaining(mockT, nil, "the gopher always has more answers"))
}

func TestHasOptionsMismatch(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, false, assertanswer.HasOptions(mockT, &slackscot.Answer{Text: "this is my final answer", Options: []slackscot.AnswerOption{slackscot.AnswerInThread()}}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "true"}))
}

func TestHasOptionsMissingOne(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, false, assertanswer.HasOptions(mockT, &slackscot.Answer{Text: "this is my final answer", Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithBroadcast()}}, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}))
}

func TestHasOptionsMatch(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, true, assertanswer.HasOptions(mockT, &slackscot.Answer{Text: "this is my final answer", Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithBroadcast()}}, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "true"}))
}

func TestHasOptionsNilAnswer(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, false, assertanswer.HasOptions(mockT, nil))
}
