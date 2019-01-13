package plugins_test

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/v2/plugins"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMatchFrequency(t *testing.T) {
	pc := viper.New()
	// With a frequency of 2, every other timestamp should match (no whitelist defined means that all channels are enabled)
	pc.Set("frequency", 2)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]
	matches := 0
	ts := "15468332%02d.036900"

	for i := 0; i < 10; i++ {
		msgt := fmt.Sprintf(ts, i)
		if h.Match("text", &slack.Msg{Timestamp: msgt}) {
			matches = matches + 1
		}
	}

	assert.Equal(t, 5, matches)
}

func TestChannelWhitelisting(t *testing.T) {
	pc := viper.New()
	// With a frequency of 1, every message should match if whitelist is on
	pc.Set("frequency", 1)
	pc.Set("channelIds", "channel1,channel2")

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]

	assert.True(t, h.Match("text", &slack.Msg{Channel: "channel1", Timestamp: "1546833210.036900"}))
	assert.True(t, h.Match("text", &slack.Msg{Channel: "channel2", Timestamp: "1546833210.036900"}))
	assert.False(t, h.Match("text", &slack.Msg{Channel: "channel3", Timestamp: "1546833210.036900"}))
}

func TestMatchConsistentWithSameTimestamp(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 2)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]

	for i := 0; i < 100; i++ {
		assert.True(t, h.Match("text", &slack.Msg{Timestamp: "1546833210.036900"}))
		assert.False(t, h.Match("text", &slack.Msg{Timestamp: "1546833222.036900"}))
	}
}

func TestQuotingOfSingleLongWord(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 1)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]

	assert.Equal(t, "\"belong\"", h.Answer(&slack.Msg{Text: "Do I belong or not?", Timestamp: "1546833210.036900"}))
}

func TestConsistentWordQuotingWithSameTimestamp(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 10)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]

	// Validate one pick with a different timestamp
	assert.Equal(t, "\"breathed\"", h.Answer(&slack.Msg{Text: `It's just a bad movie, where there's no crying. Handing the keys to me in this Red Lion. 
			Where the lock that you locked in the suite says there's no prying. When the breath that you breathed in 
			the street screams there's no science`, Timestamp: "1546833310.036900"}))

	// Validate that calling the answer function a hundred times with the same timestamp results in the same pick
	for i := 0; i < 100; i++ {
		assert.Equal(t, "\"locked\"", h.Answer(&slack.Msg{Text: `It's just a bad movie, where there's no crying. Handing the keys to me in this Red Lion. 
			Where the lock that you locked in the suite says there's no prying. When the breath that you breathed in 
			the street screams there's no science`, Timestamp: "1546833210.036900"}))
	}
}

func TestNoQuotingIfOnlySmallWords(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 1)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]

	assert.Equal(t, "", h.Answer(&slack.Msg{Text: "Do I or not?", Timestamp: "1546833210.036900"}))
}
