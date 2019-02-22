package assertplugin_test

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/test/assertanswer"
	"github.com/alexandre-normand/slackscot/test/assertplugin"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

type myLittleTester struct {
	slackscot.Plugin
}

func newLittleTester() (mlt *myLittleTester) {
	mlt = new(myLittleTester)
	mlt.Name = "myLittleTester"

	mlt.Commands = []slackscot.ActionDefinition{{
		Hidden: true,
		Match: func(m *slackscot.IncomingMessage) bool {
			return strings.HasPrefix(m.NormalizedText, "tell me where the black-capped chickadee is")
		},
		Usage:       "",
		Description: "",
		Answer:      findChicakee,
	}}

	mlt.HearActions = []slackscot.ActionDefinition{
		{
			Hidden: true,
			Match: func(m *slackscot.IncomingMessage) bool {
				return strings.Contains(m.NormalizedText, "are you up?")
			},
			Usage:       "",
			Description: "",
			Answer:      areYouAnswerer,
		},
		{
			Hidden: true,
			Match: func(m *slackscot.IncomingMessage) bool {
				return strings.Contains(m.NormalizedText, "hey")
			},
			Usage:       "",
			Description: "",
			Answer:      heyAnswerer,
		},
		{
			Hidden: true,
			Match: func(m *slackscot.IncomingMessage) bool {
				return strings.Contains(m.NormalizedText, "blue jays")
			},
			Usage:       "",
			Description: "",
			Answer:      mlt.emojiReact,
		},
	}
	mlt.ScheduledActions = nil

	return mlt
}

func findChicakee(m *slackscot.IncomingMessage) *slackscot.Answer {
	return &slackscot.Answer{Text: "👀 in the 🌲"}
}

func areYouAnswerer(m *slackscot.IncomingMessage) *slackscot.Answer {
	return &slackscot.Answer{Text: "I'm 😴, you?"}
}

func heyAnswerer(m *slackscot.IncomingMessage) *slackscot.Answer {
	return &slackscot.Answer{Text: "hey wut?"}
}

func (mlt *myLittleTester) emojiReact(m *slackscot.IncomingMessage) *slackscot.Answer {
	mlt.EmojiReactor.AddReaction("owl", slack.NewRefToMessage(m.Channel, m.Timestamp))

	return nil
}

func TestCommandResultNonValid(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New("bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, false, assertplugin.AnswersAndReacts(mockT, &myLittleTester.Plugin, &slack.Msg{Text: "<@bot> tell me where the black-capped chickadee is"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 10)
	}))
}

func TestCommandResultValid(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New("bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.AnswersAndReacts(mockT, &myLittleTester.Plugin, &slack.Msg{Text: "<@bot> tell me where the black-capped chickadee is"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "👀 in the 🌲")
	}))
}

func TestHearResultValid(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New("bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.AnswersAndReacts(mockT, &myLittleTester.Plugin, &slack.Msg{Text: "are you up?"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "I'm 😴, you?")
	}))
}

func TestDirectCommandResultValid(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New("bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.AnswersAndReacts(mockT, &myLittleTester.Plugin, &slack.Msg{Text: "tell me where the black-capped chickadee is", Channel: "DTOTHEBOT"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "👀 in the 🌲")
	}))
}

func TestEmojiReaction(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New("bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.AnswersAndReacts(mockT, &myLittleTester.Plugin, &slack.Msg{Text: "blue jays"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers) && assert.Contains(t, emojis, "owl")
	}))
}

func TestMultipleAnswersWithEmojiReaction(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New("bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.AnswersAndReacts(mockT, &myLittleTester.Plugin, &slack.Msg{Text: "hey, are you up? I think I just saw blue jays"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 2) && assertanswer.HasText(t, answers[0], "I'm 😴, you?") && assertanswer.HasText(t, answers[1], "hey wut?") && assert.Contains(t, emojis, "owl")
	}))
}
