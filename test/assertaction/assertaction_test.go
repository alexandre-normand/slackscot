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

type emojiReactionTester struct {
	slackscot.Plugin
	emojis           []string
	channelTargets   []string
	timestampTargets []string
}

func newEmojiReactionTester(emojis ...string) (e *emojiReactionTester) {
	return newEmojiReactionTesterWithTarget(nil, nil, emojis...)
}

func newEmojiReactionTesterWithTarget(channelsToReactTo []string, timestampsToReactTo []string, emojis ...string) (e *emojiReactionTester) {
	e = new(emojiReactionTester)
	e.Name = "emojiReactionTester"

	e.HearActions = []slackscot.ActionDefinition{{
		Hidden: true,
		Match: func(m *slackscot.IncomingMessage) bool {
			return strings.Contains(m.NormalizedText, "blue jays")
		},
		Usage:       "🦉 make me happy",
		Description: "",
		Answer:      e.emojiReact,
	}}
	e.ScheduledActions = nil
	e.emojis = emojis
	e.channelTargets = channelsToReactTo
	e.timestampTargets = timestampsToReactTo

	return e
}

func (e *emojiReactionTester) emojiReact(m *slackscot.IncomingMessage) *slackscot.Answer {

	for i, emoji := range e.emojis {
		channel := m.Channel
		timestamp := m.Timestamp

		// Override if we have a target channel / target timestamp for that index
		if len(e.channelTargets) > i {
			channel = e.channelTargets[i]
		}

		if len(e.timestampTargets) > i {
			timestamp = e.timestampTargets[i]
		}

		e.EmojiReactor.AddReaction(emoji, slack.NewRefToMessage(channel, timestamp))
	}

	return nil
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

func TestAssertMatchAndEmojiReactNoMatch(t *testing.T) {
	mockT := new(testing.T)
	e := newEmojiReactionTester("cat", "thumbsup")

	assert.Equal(t, false, assertaction.MatchesAndEmojiReacts(mockT, &e.Plugin, e.HearActions[0], &slackscot.IncomingMessage{NormalizedText: "ping", Msg: slack.Msg{Text: "ping", Channel: "birds", Timestamp: "12345"}}))
}

func TestAssertMatchAndEmojiReactWithMatch(t *testing.T) {
	mockT := new(testing.T)
	e := newEmojiReactionTester("cat", "thumbsup")

	assert.Equal(t, true, assertaction.MatchesAndEmojiReacts(mockT, &e.Plugin, e.HearActions[0], &slackscot.IncomingMessage{NormalizedText: "blue jays are cool", Msg: slack.Msg{Text: "blue jays are cool", Channel: "birds", Timestamp: "12345"}}, "cat", "thumbsup"))
}

func TestAssertMatchAndEmojiReactWithWrongChannel(t *testing.T) {
	mockT := new(testing.T)
	e := newEmojiReactionTesterWithTarget([]string{"otherChan", "otherChan"}, []string{"otherTs", "otherTs"}, "cat", "thumbsup")

	assert.Equal(t, false, assertaction.MatchesAndEmojiReacts(mockT, &e.Plugin, e.HearActions[0], &slackscot.IncomingMessage{NormalizedText: "blue jays are cool", Msg: slack.Msg{Text: "blue jays are cool", Channel: "birds", Timestamp: "12345"}}, "cat", "thumbsup"))
}

func TestAssertMatchAndEmojiReactWithManyChannels(t *testing.T) {
	mockT := new(testing.T)
	e := newEmojiReactionTesterWithTarget([]string{"birds", "otherChan"}, []string{"12345", "otherTs"}, "cat", "thumbsup")

	assert.Equal(t, false, assertaction.MatchesAndEmojiReacts(mockT, &e.Plugin, e.HearActions[0], &slackscot.IncomingMessage{NormalizedText: "blue jays are cool", Msg: slack.Msg{Text: "blue jays are cool", Channel: "birds", Timestamp: "12345"}}, "cat", "thumbsup"))
}
