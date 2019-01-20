package plugins_test

import (
	"github.com/alexandre-normand/slackscot/v2/plugins"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEmojiBannerTrigger(t *testing.T) {
	pc := viper.New()

	ebm, err := plugins.NewEmojiBannerMaker(pc)
	assert.Nil(t, err)

	defer ebm.Close()

	c := ebm.Commands[0]

	assert.Equal(t, false, c.Match("other", &slack.Msg{}))
	assert.Equal(t, false, c.Match("emoji", &slack.Msg{}))
	assert.Equal(t, true, c.Match("emoji banner cats :cat:", &slack.Msg{}))
}

func TestEmojiBannerGenerationWithDefaultFont(t *testing.T) {
	pc := viper.New()

	ebm, err := plugins.NewEmojiBannerMaker(pc)
	assert.Nil(t, err)

	defer ebm.Close()

	c := ebm.Commands[0]

	assert.Equal(t, "\r\n⬜️⬜️:cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️⬜️:cat::cat::cat:⬜️⬜️⬜️⬜️⬜️⬜️:cat:"+
		":cat::cat::cat::cat::cat::cat::cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️:cat::cat::cat::cat::cat::cat:"+
		":cat::cat:\n⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️"+
		":cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat:\n:cat:⬜️⬜️:cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️"+
		":cat:⬜️⬜️⬜️⬜️:cat::cat::cat::cat::cat:⬜️⬜️:cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️:cat:⬜️⬜️⬜️:cat::cat::cat::cat:"+
		":cat::cat:\n:cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️:cat::cat::cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️⬜️"+
		"⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️\n:cat:⬜️⬜️:cat::cat::cat::cat::cat::cat:⬜️:cat:⬜️⬜️:cat::cat::cat::cat::cat:"+
		"⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️:cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️:cat:⬜️⬜️⬜️\n⬜️:cat::cat::cat::cat:"+
		":cat::cat::cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️:cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️:cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️:cat::cat::cat:"+
		":cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️⬜️\n⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️"+
		"⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️\n", c.Answer(&slack.Msg{Text: "emoji banner cats :cat:"}))
}

func TestEmojiBannerGenerationWithBannerFont(t *testing.T) {
	pc := viper.New()
	pc.Set("figletFontUrl", "http://www.figlet.org/fonts/banner.flf")

	ebm, err := plugins.NewEmojiBannerMaker(pc)
	assert.Nil(t, err)

	defer ebm.Close()

	c := ebm.Commands[0]

	assert.Equal(t, "\r\n⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️\n⬜️⬜️:cat::cat::cat::cat:⬜️⬜️⬜️⬜️⬜️:cat:"+
		":cat:⬜️⬜️⬜️⬜️:cat::cat::cat::cat::cat:⬜️⬜️⬜️:cat::cat::cat::cat:⬜️⬜️\n⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️:cat:"+
		"⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️\n⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️:cat::cat::cat::cat:⬜️⬜️\n⬜️:cat:"+
		"⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat::cat::cat::cat::cat::cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️:cat:⬜️\n⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:"+
		"⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️\n⬜️⬜️:cat::cat::cat::cat:⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️:cat:⬜️⬜️⬜️⬜️⬜️:cat::cat::cat:"+
		":cat:⬜️⬜️\n⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️\n", c.Answer(&slack.Msg{Text: "emoji banner cats :cat:"}))
}

func TestBadFontURLShouldFailPluginCreation(t *testing.T) {
	pc := viper.New()
	pc.Set("figletFontUrl", "https://invalid.url.is.bad/")

	_, err := plugins.NewEmojiBannerMaker(pc)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "Error loading font url")
	}
}

func TestInvalidFontURLShouldFailPluginCreation(t *testing.T) {
	pc := viper.New()
	pc.Set("figletFontUrl", "%proto:")

	_, err := plugins.NewEmojiBannerMaker(pc)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "Invalid font url")
	}
}
