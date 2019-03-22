package plugins_test

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/test/assertanswer"
	"github.com/alexandre-normand/slackscot/test/assertplugin"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEmojiBannerTrigger(t *testing.T) {
	pc := viper.New()

	ebm, err := plugins.NewEmojiBannerMaker(pc)
	assert.NoError(t, err)
	defer ebm.Close()

	assertplugin := assertplugin.New(t, "robert")

	assertplugin.AnswersAndReacts(&ebm.Plugin, &slack.Msg{Text: "<@robert> other"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers)
	})

	assertplugin.AnswersAndReacts(&ebm.Plugin, &slack.Msg{Text: "<@robert> emoji"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers)
	})

	assertplugin.AnswersAndReacts(&ebm.Plugin, &slack.Msg{Text: "<@robert> emoji banner cat :cat:"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\r\n⬜️⬜️:cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️⬜️:cat::cat::cat:⬜️⬜️⬜️⬜️⬜️⬜️:cat::cat::cat:"+
			":cat::cat::cat::cat::cat::cat::cat::cat::cat::cat:\n⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat:\n:cat:⬜️⬜️:cat:"+
			":cat::cat::cat::cat::cat:⬜️⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat::cat::cat::cat::cat:⬜️⬜️:cat::cat::cat::cat::cat::cat:\n:cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️"+
			":cat::cat::cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️\n:cat:⬜️⬜️:cat::cat::cat::cat::cat::cat:⬜️:cat:⬜️⬜️:cat::cat::cat::cat::cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️"+
			":cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️\n⬜️:cat::cat::cat::cat::cat::cat::cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️:cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️:cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️\n"+
			"⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️\n")
	})
}

func TestEmojiBannerGenerationWithWrongUsage(t *testing.T) {
	pc := viper.New()

	ebm, err := plugins.NewEmojiBannerMaker(pc)
	assert.NoError(t, err)
	defer ebm.Close()

	assertplugin := assertplugin.New(t, "robert")

	assertplugin.AnswersAndReacts(&ebm.Plugin, &slack.Msg{Text: "<@robert> emoji banner"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "`Wrong usage`: emoji banner `<word of 4 characters or less>` `<emoji>`")
	})

	assertplugin.AnswersAndReacts(&ebm.Plugin, &slack.Msg{Text: "<@robert> emoji banner cats"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "`Wrong usage`: emoji banner `<word of 4 characters or less>` `<emoji>`")
	})
}

func TestEmojiBannerGenerationWithLongWord(t *testing.T) {
	pc := viper.New()

	ebm, err := plugins.NewEmojiBannerMaker(pc)
	assert.NoError(t, err)
	defer ebm.Close()

	assertplugin := assertplugin.New(t, "robert")

	assertplugin.AnswersAndReacts(&ebm.Plugin, &slack.Msg{Text: "<@robert> emoji banner hello :bug:"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "`Wrong usage` (word *longer* than `4` characters): emoji banner `<word of 5 characters or less>` `<emoji>`")
	})
}

func TestEmojiBannerGenerationWithDefaultFont(t *testing.T) {
	pc := viper.New()

	ebm, err := plugins.NewEmojiBannerMaker(pc)
	assert.NoError(t, err)
	defer ebm.Close()

	assertplugin := assertplugin.New(t, "robert")

	assertplugin.AnswersAndReacts(&ebm.Plugin, &slack.Msg{Text: "<@robert> emoji banner cat :cat:"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\r\n⬜️⬜️:cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️⬜️:cat::cat::cat:⬜️⬜️⬜️⬜️⬜️⬜️:cat::cat::cat:"+
			":cat::cat::cat::cat::cat::cat::cat::cat::cat::cat:\n⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat:\n:cat:⬜️⬜️:cat:"+
			":cat::cat::cat::cat::cat:⬜️⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat::cat::cat::cat::cat:⬜️⬜️:cat::cat::cat::cat::cat::cat:\n:cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️"+
			":cat::cat::cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️\n:cat:⬜️⬜️:cat::cat::cat::cat::cat::cat:⬜️:cat:⬜️⬜️:cat::cat::cat::cat::cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️"+
			":cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️\n⬜️:cat::cat::cat::cat::cat::cat::cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️:cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️:cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️\n"+
			"⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️\n")
	})
}

func TestEmojiBannerGenerationWithBannerFont(t *testing.T) {
	pc := viper.New()
	pc.Set("figletFontUrl", "http://www.figlet.org/fonts/banner.flf")

	ebm, err := plugins.NewEmojiBannerMaker(pc)
	assert.NoError(t, err)
	defer ebm.Close()

	assertplugin := assertplugin.New(t, "robert")

	assertplugin.AnswersAndReacts(&ebm.Plugin, &slack.Msg{Text: "<@robert> emoji banner cat :cat:"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\r\n⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️\n⬜️⬜️:cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️:cat:"+
			":cat:⬜️⬜️⬜️⬜️:cat::cat::cat::cat::cat:⬜️\n⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️\n⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️"+
			":cat:⬜️⬜️⬜️\n⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️\n⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️\n"+
			"⬜️⬜️:cat::cat::cat::cat:⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️\n⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️\n")
	})
}

func TestBadFontURLShouldFailPluginCreation(t *testing.T) {
	pc := viper.New()
	pc.Set("figletFontUrl", "https://invalid.url.is.bad/")

	_, err := plugins.NewEmojiBannerMaker(pc)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Error loading font url")
	}
}

func TestInvalidFontURLShouldFailPluginCreation(t *testing.T) {
	pc := viper.New()
	pc.Set("figletFontUrl", "%proto:")

	_, err := plugins.NewEmojiBannerMaker(pc)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Invalid font url")
	}
}
