package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"math/rand"
	"strconv"
	"strings"
	"unicode"
)

const (
	channelIDsKey        = "channelIDs"
	ignoredChannelIDsKey = "ignoredChannelIDs"
	frequencyKey         = "frequency"
)

const (
	// FingerQuoterPluginName holds identifying name for the finger quoter plugin
	FingerQuoterPluginName = "fingerQuoter"
)

// FingerQuoter holds the plugin data for the finger quoter plugin
type FingerQuoter struct {
	slackscot.Plugin
	channels        []string
	ignoredChannels []string
	frequency       int
}

// NewFingerQuoter creates a new instance of the plugin
func NewFingerQuoter(config *config.PluginConfig) (f *FingerQuoter, err error) {
	if ok := config.IsSet(frequencyKey); !ok {
		return nil, fmt.Errorf("Missing %s config key: %s", FingerQuoterPluginName, frequencyKey)
	}

	f = new(FingerQuoter)
	f.channels = config.GetStringSlice(channelIDsKey)
	f.ignoredChannels = config.GetStringSlice(ignoredChannelIDsKey)
	f.frequency = config.GetInt(frequencyKey)
	f.Name = FingerQuoterPluginName
	f.HearActions = []slackscot.ActionDefinition{{
		Hidden: true,
		// Match based on the frequency probability and whether or not the channel is whitelisted
		Match:       f.trigger,
		Usage:       "just converse",
		Description: "finger quoter listens to what people say and (sometimes) finger quotes a word",
		Answer:      f.fingerQuoteMsg,
	}}

	return f, err
}

func (f *FingerQuoter) trigger(m *slackscot.IncomingMessage) bool {
	if !isChannelEnabled(m.Channel, f.channels, f.ignoredChannels) {
		return false
	}

	ts, err := strconv.ParseFloat(m.Timestamp, 64)
	if err != nil {
		f.Logger.Debugf("[%s] Skipping message [%v] because of error converting timestamp to float: %v\n", FingerQuoterPluginName, m, err)
		return false
	}

	fullTs := ts * 1000000.

	// Make the random generator use a seed based on the message id so that we preserve the same matches when messages get updated
	randomGen := rand.New(rand.NewSource(int64(fullTs)))

	// Determine if we're going to react this time or not
	return randomGen.Int31n(int32(f.frequency)) == 0
}

func (f *FingerQuoter) fingerQuoteMsg(m *slackscot.IncomingMessage) *slackscot.Answer {
	candidates := splitInputIntoWordsLongerThan(m.Text, 4)

	if len(candidates) > 0 {
		ts, err := strconv.ParseFloat(m.Timestamp, 64)
		if err != nil {
			f.Logger.Debugf("[%s] Skipping message [%v] because of error converting timestamp to float: %v\n", FingerQuoterPluginName, m, err)
		} else {
			fullTs := ts * 1000000.

			// Make the random generator use a seed based on the message id so that we preserve the same matches when messages get updated
			randomGen := rand.New(rand.NewSource(int64(fullTs)))

			i := randomGen.Int31n(int32(len(candidates)))
			return &slackscot.Answer{Text: fmt.Sprintf("\"%s\"", candidates[i])}
		}
	}

	// Not this time friends, skip it
	return nil
}

func splitInputIntoWordsLongerThan(t string, minLen int) []string {
	words := strings.FieldsFunc(t, func(c rune) bool {
		return !unicode.IsLetter(c) && c != '-'
	})

	return filterWordsLongerThan(words, minLen)
}

func filterWordsLongerThan(words []string, minLen int) []string {
	candidates := make([]string, 0)
	for _, w := range words {
		if len(w) > minLen {
			candidates = append(candidates, w)
		}
	}

	return candidates
}

func isChannelEnabled(channelID string, whitelist []string, ignoredChannels []string) bool {
	if isChannelWhiteListed(channelID, whitelist) && !isChannelIgnored(channelID, ignoredChannels) {
		return true
	}

	return false
}

func isChannelIgnored(channelID string, ignoredChannels []string) bool {
	for _, c := range ignoredChannels {
		if c == channelID {
			return true
		}
	}

	return false
}

func isChannelWhiteListed(channelID string, whitelist []string) bool {
	// Default to all channels whitelisted if none specified which is either that the element is missing or if the only element is empty string
	if len(whitelist) == 0 || (len(whitelist) == 1 && whitelist[0] == "") {
		return true
	}

	for _, c := range whitelist {
		if c == channelID {
			return true
		}
	}

	return false
}
