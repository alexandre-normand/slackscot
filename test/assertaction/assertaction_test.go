package assertaction_test

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/test/assertaction"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

var echoAction = slackscot.ActionDefinition{
	Hidden: false,
	Match: func(m *slackscot.IncomingMessage) bool {
		return strings.Contains(m.NormalizedText, "ping")
	},
	Usage:       "ping",
	Description: "Sends `pong` on hearing `ping`",
	Answer: func(m *slackscot.IncomingMessage) *slackscot.Answer {
		return &slackscot.Answer{Text: "pong"}
	},
}

func TestAssertNoMatchWhenMatch(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, false, assertaction.NotMatch(mockT, echoAction, &slackscot.IncomingMessage{NormalizedText: "ping", Msg: slack.Msg{Text: "ping"}}))
}

func TestAssertNoMatchWhenNoMatch(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, true, assertaction.NotMatch(mockT, echoAction, &slackscot.IncomingMessage{NormalizedText: "pang", Msg: slack.Msg{Text: "pang"}}))
}

func TestAssertMatchAndAnswersWhenNoMatch(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, false, assertaction.MatchesAndAnswers(mockT, echoAction, &slackscot.IncomingMessage{NormalizedText: "pang", Msg: slack.Msg{Text: "pang"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return true
	}))
}

func TestAssertMatchAndAnswersWhenMatchesButAnswerNotValid(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, false, assertaction.MatchesAndAnswers(mockT, echoAction, &slackscot.IncomingMessage{NormalizedText: "ping", Msg: slack.Msg{Text: "ping"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return false
	}))
}

func TestAssertMatchAndAnswersWhenMatchesWithValidAnswer(t *testing.T) {
	mockT := new(testing.T)
	assert.Equal(t, true, assertaction.MatchesAndAnswers(mockT, echoAction, &slackscot.IncomingMessage{NormalizedText: "ping", Msg: slack.Msg{Text: "ping"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return true
	}))
}
