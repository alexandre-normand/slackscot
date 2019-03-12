package plugins_test

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/test/assertanswer"
	"github.com/alexandre-normand/slackscot/test/assertplugin"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSendValidVersionMessage(t *testing.T) {
	v := plugins.NewVersionner("little-red", "1.0.0")
	assert.NotNil(t, v)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&v.Plugin, &slack.Msg{Text: "<@bot> version"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "I'm `little-red`, version `1.0.0`")
	})
}

func TestMatchOnVersionCommand(t *testing.T) {
	v := plugins.NewVersionner("little-red", "1.0.0")
	assert.NotNil(t, v)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&v.Plugin, &slack.Msg{Text: "<@bot> version"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1)
	})

	assertplugin.AnswersAndReacts(&v.Plugin, &slack.Msg{Text: "<@bot>  version"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 0)
	})

	assertplugin.AnswersAndReacts(&v.Plugin, &slack.Msg{Text: "<@bot> version "}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1)
	})
}
