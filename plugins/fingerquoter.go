package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/v2"
	"github.com/alexandre-normand/slackscot/v2/config"
	"github.com/nlopes/slack"
	"math/rand"
	"strconv"
	"strings"
	"unicode"
)

const (
	channelIdsKey = "channelIds"
	frequencyKey  = "frequency"
)

const (
	// FingerQuoterPluginName holds identifying name for the finger quoter plugin
	FingerQuoterPluginName = "fingerQuoter"
)

// FingerQuoter holds the plugin data for the finger quoter plugin
type FingerQuoter struct {
	slackscot.Plugin
	channels  []string
	frequency int
}

// NewFingerQuoter creates a new instance of the plugin
func NewFingerQuoter(config *config.PluginConfig) (f *FingerQuoter, err error) {
	if ok := config.IsSet(frequencyKey); !ok {
		return nil, fmt.Errorf("Missing %s config key: %s", FingerQuoterPluginName, frequencyKey)
	}

	f = new(FingerQuoter)
	channelValue := config.GetString(channelIdsKey)
	f.channels = strings.Split(channelValue, ",")
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

func (f *FingerQuoter) trigger(t string, m *slack.Msg) bool {
	if !isChannelWhiteListed(m.Channel, f.channels) {
		return false
	}

	ts, err := strconv.ParseFloat(m.Timestamp, 64)
	if err != nil {
		f.Logger.Debugf("[%s] Skipping message [%v] because of error converting timestamp to float: %v\n", FingerQuoterPluginName, m, err)
	} else {
		// Make the random generator use a seed based on the message id so that we preserve the same matches when messages get updated
		randomGen := rand.New(rand.NewSource(int64(ts)))

		// Determine if we're going to react this time or not
		return randomGen.Int31n(int32(f.frequency)) == 0
	}
	return false
}

func (f *FingerQuoter) fingerQuoteMsg(m *slack.Msg) string {
	candidates := splitInputIntoWordsLongerThan(m.Text, 4)

	if len(candidates) > 0 {
		ts, err := strconv.ParseFloat(m.Timestamp, 64)
		if err != nil {
			f.Logger.Debugf("[%s] Skipping message [%v] because of error converting timestamp to float: %v\n", FingerQuoterPluginName, m, err)
		} else {
			// Make the random generator use a seed based on the message id so that we preserve the same matches when messages get updated
			randomGen := rand.New(rand.NewSource(int64(ts)))

			i := randomGen.Int31n(int32(len(candidates)))
			return fmt.Sprintf("\"%s\"", candidates[i])
		}
	}

	// Not this time, skip
	return ""
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

func isChannelWhiteListed(channelId string, whitelist []string) bool {
	// Default to all channels whitelisted if none specified which is either that the element is missing or if the only element is empty string
	if len(whitelist) == 0 {
		return true
	}

	for _, c := range whitelist {
		if c == channelId {
			return true
		}
	}

	return false
}
