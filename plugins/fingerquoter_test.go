package plugins_test

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"testing"
)

func TestMissingFrequencyConfig(t *testing.T) {
	pc := viper.New()

	_, err := plugins.NewFingerQuoter(pc)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "Missing fingerQuoter config key: frequency")
	}
}

func TestMatchFrequency(t *testing.T) {
	pc := viper.New()
	// With a frequency of 2, every other timestamp should match (no whitelist defined means that all channels are enabled)
	pc.Set("frequency", 2)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]
	matches := 0
	ts := "1546833245.0369%02d"

	for i := 0; i < 10; i++ {
		msgt := fmt.Sprintf(ts, i)
		if h.Match(&slackscot.IncomingMessage{NormalizedText: "text", Msg: slack.Msg{Timestamp: msgt}}) {
			matches = matches + 1
		}
	}

	assert.Equal(t, 5, matches)
}

func TestChannelWhitelisting(t *testing.T) {
	pc := viper.New()
	// With a frequency of 1, every message should match if whitelist is on
	pc.Set("frequency", 1)
	pc.Set("channelIds", []string{"channel1", "channel2"})

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]

	assert.True(t, h.Match(&slackscot.IncomingMessage{NormalizedText: "text", Msg: slack.Msg{Channel: "channel1", Timestamp: "1546833210.036900"}}))
	assert.True(t, h.Match(&slackscot.IncomingMessage{NormalizedText: "text", Msg: slack.Msg{Channel: "channel2", Timestamp: "1546833210.036900"}}))
	assert.False(t, h.Match(&slackscot.IncomingMessage{NormalizedText: "text", Msg: slack.Msg{Channel: "channel3", Timestamp: "1546833210.036900"}}))
}

func TestDefaultWhitelistingEnablesForAll(t *testing.T) {
	pc := viper.New()
	// With a frequency of 1, every message should match if whitelist is on
	pc.Set("frequency", 1)
	pc.Set("channelIds", "")

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]

	assert.True(t, h.Match(&slackscot.IncomingMessage{NormalizedText: "text", Msg: slack.Msg{Channel: "channel1", Timestamp: "1546833210.036900"}}))
	assert.True(t, h.Match(&slackscot.IncomingMessage{NormalizedText: "text", Msg: slack.Msg{Channel: "channel2", Timestamp: "1546833210.036900"}}))
	assert.True(t, h.Match(&slackscot.IncomingMessage{NormalizedText: "text", Msg: slack.Msg{Channel: "channel3", Timestamp: "1546833210.036900"}}))
}

func TestMatchConsistentWithSameTimestamp(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 2)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]

	for i := 0; i < 100; i++ {
		assert.True(t, h.Match(&slackscot.IncomingMessage{NormalizedText: "text", Msg: slack.Msg{Timestamp: "1546833210.036903"}}))
		assert.False(t, h.Match(&slackscot.IncomingMessage{NormalizedText: "text", Msg: slack.Msg{Timestamp: "1546833222.031904"}}))
	}
}

func TestMatchFalseWhenCorruptedTimestamp(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 1)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	// Set debug logger
	var b strings.Builder
	f.Logger = slackscot.NewSLogger(log.New(&b, "", 0), true)

	h := f.HearActions[0]

	assert.False(t, h.Match(&slackscot.IncomingMessage{NormalizedText: "text", Msg: slack.Msg{Channel: "channel1", Timestamp: "NotAFloatValue"}}))
	assert.Contains(t, b.String(), "error converting timestamp to float")
}

func TestNoAnswerWhenCorruptedTimestamp(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 1)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	// Set debug logger
	var b strings.Builder
	f.Logger = slackscot.NewSLogger(log.New(&b, "", 0), true)

	h := f.HearActions[0]

	assert.Equal(t, "", h.Answer(&slackscot.IncomingMessage{Msg: slack.Msg{Channel: "channel1", Timestamp: "NotAFloatValue", Text: "This is a text with longer and shorter words"}}).Text)
	assert.Contains(t, b.String(), "error converting timestamp to float")
}

func TestQuotingOfSingleLongWord(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 1)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]

	assert.Equal(t, "\"belong\"", h.Answer(&slackscot.IncomingMessage{Msg: slack.Msg{Text: "Do I belong or not?", Timestamp: "1546833210.036900"}}).Text)
}

func TestConsistentWordQuotingWithSameTimestamp(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 10)
	pc.Set("channelIds", "")

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]

	// Validate one pick with a different timestamp
	assert.Equal(t, "\"screams\"", h.Answer(&slackscot.IncomingMessage{Msg: slack.Msg{Text: `It's just a bad movie, where there's no crying. Handing the keys to me in this Red Lion. 
			Where the lock that you locked in the suite says there's no prying. When the breath that you breathed in 
			the street screams there's no science`, Timestamp: "1546833310.036900"}}).Text)

	// Validate that calling the answer function a hundred times with the same timestamp results in the same pick
	for i := 0; i < 100; i++ {
		assert.Equal(t, "\"Where\"", h.Answer(&slackscot.IncomingMessage{Msg: slack.Msg{Text: `It's just a bad movie, where there's no crying. Handing the keys to me in this Red Lion. 
			Where the lock that you locked in the suite says there's no prying. When the breath that you breathed in 
			the street screams there's no science`, Timestamp: "1546833210.036900"}}).Text)
	}

	// Validate that a timestamp *almost* equal to the prior one (except for decimals) results in something different to make sure
	// we don't ignore those
	assert.Equal(t, "\"Handing\"", h.Answer(&slackscot.IncomingMessage{Msg: slack.Msg{Text: `It's just a bad movie, where there's no crying. Handing the keys to me in this Red Lion. 
			Where the lock that you locked in the suite says there's no prying. When the breath that you breathed in 
			the street screams there's no science`, Timestamp: "1546833210.036907"}}).Text)
}

func TestNoQuotingIfOnlySmallWords(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 1)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	h := f.HearActions[0]

	assert.Equal(t, "", h.Answer(&slackscot.IncomingMessage{Msg: slack.Msg{Text: "Do I or not?", Timestamp: "1546833210.036900"}}).Text)
}
