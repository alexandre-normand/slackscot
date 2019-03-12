package plugins_test

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/test/assertanswer"
	"github.com/alexandre-normand/slackscot/test/assertplugin"
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
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Missing fingerQuoter config key: frequency")
	}
}

func TestMatchFrequency(t *testing.T) {
	pc := viper.New()
	// With a frequency of 2, every other timestamp should match (no whitelist defined means that all channels are enabled)
	pc.Set("frequency", 2)

	f, err := plugins.NewFingerQuoter(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")

	matches := 0
	ts := "1546833245.0369%02d"
	for i := 0; i < 10; i++ {
		msgt := fmt.Sprintf(ts, i)

		assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Timestamp: msgt}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			matches = matches + len(answers)
			return true
		})
	}

	assert.Equal(t, 5, matches)
}

func TestChannelWhitelisting(t *testing.T) {
	pc := viper.New()
	// With a frequency of 1, every message should match if whitelist is on
	pc.Set("frequency", 1)
	pc.Set("channelIDs", []string{"channel1", "channel2"})

	f, err := plugins.NewFingerQuoter(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel1", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\"thing\"")
	})

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel2", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\"thing\"")
	})

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel3", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers)
	})
}

func TestChannelIgnoring(t *testing.T) {
	pc := viper.New()
	// With a frequency of 1, every message should match if whitelist is on
	pc.Set("frequency", 1)
	pc.Set("channelIDs", []string{"channel1", "channel2"})
	pc.Set("ignoredChannelIDs", []string{"channel2"})

	f, err := plugins.NewFingerQuoter(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel1", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\"thing\"")
	})

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel2", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers)
	})

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel3", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers)
	})
}

func TestChannelIgnoredWithDefaultWhitelisting(t *testing.T) {
	pc := viper.New()
	// With a frequency of 1, every message should match if whitelist is on
	pc.Set("frequency", 1)
	pc.Set("channelIDs", "")
	pc.Set("ignoredChannelIDs", []string{"channel2"})

	f, err := plugins.NewFingerQuoter(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel1", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\"thing\"")
	})

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel2", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers)
	})

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel3", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\"thing\"")
	})
}

func TestDefaultWhitelistingEnablesForAll(t *testing.T) {
	pc := viper.New()
	// With a frequency of 1, every message should match if whitelist is on
	pc.Set("frequency", 1)
	pc.Set("channelIDs", "")

	f, err := plugins.NewFingerQuoter(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel1", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\"thing\"")
	})

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel2", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\"thing\"")
	})

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel3", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\"thing\"")
	})
}

func TestMatchConsistentWithSameTimestamp(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 2)

	f, err := plugins.NewFingerQuoter(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")

	for i := 0; i < 100; i++ {
		assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel1", Timestamp: "1546833210.036903"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1)
		})

		assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel1", Timestamp: "1546833222.031904"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, answers)
		})
	}
}

func TestMatchFalseWhenCorruptedTimestamp(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 1)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	// Set debug logger
	var b strings.Builder
	logger := log.New(&b, "", 0)

	assertplugin := assertplugin.New(t, "bot", assertplugin.OptionLog(logger))

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "some random thing someone could say", Channel: "channel1", Timestamp: "NotAFloatValue"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers) && assert.Contains(t, b.String(), "error converting timestamp to float")
	})
}

func TestNoAnswerWhenCorruptedTimestamp(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 1)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	// Attach logger to plugin
	var b strings.Builder
	logger := log.New(&b, "", 0)

	assertplugin := assertplugin.New(t, "bot", assertplugin.OptionLog(logger))

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "This is a text with longer and shorter words", Timestamp: "NotAFloatValue"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers) && assert.Contains(t, b.String(), "error converting timestamp to float")
	})
}

func TestQuotingOfSingleLongWord(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 1)

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "Do I belong or not?", Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\"belong\"")
	})
}

func TestNotQuotingPartsOfURLs(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 1)

	f, err := plugins.NewFingerQuoter(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "https://google.com/query?bigfoot=friend", Timestamp: "1546833310.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers)
	})
}

func TestConsistentWordQuotingWithSameTimestamp(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 1)
	pc.Set("channelIDs", "")

	f, err := plugins.NewFingerQuoter(pc)
	assert.Nil(t, err)

	assertplugin := assertplugin.New(t, "bot")

	// Validate one pick with a different timestamp
	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: `It's just a bad movie, where there's no crying. Handing the keys to me in this Red Lion. 
			Where the lock that you locked in the suite says there's no prying. When the breath that you breathed in 
			the street screams there's no science`, Timestamp: "1546833315.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\"breath\"")
	})

	// Validate that calling the answer function a hundred times with the same timestamp results in the same pick
	for i := 0; i < 100; i++ {
		if !assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: `It's just a bad movie, where there's no crying. Handing the keys to me in this Red Lion. 
			Where the lock that you locked in the suite says there's no prying. When the breath that you breathed in 
			the street screams there's no science`, Timestamp: "1546833210.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\"street\"")
		}) {
			break
		}
	}

	// Validate that a timestamp *almost* equal to the prior one (except for decimals) results in something different to make sure
	// we don't ignore those
	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: `It's just a bad movie, where there's no crying. Handing the keys to me in this Red Lion. 
			Where the lock that you locked in the suite says there's no prying. When the breath that you breathed in 
			the street screams there's no science`, Timestamp: "1546833210.036907"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "\"Where\"")
	})
}

func TestNoQuotingIfOnlySmallWords(t *testing.T) {
	pc := viper.New()
	pc.Set("frequency", 1)

	f, err := plugins.NewFingerQuoter(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&f.Plugin, &slack.Msg{Text: "Do I or not?", Timestamp: "1546833310.036900"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers)
	})
}
