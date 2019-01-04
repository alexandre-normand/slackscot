package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/nlopes/slack"
	"math/rand"
	"strconv"
	"strings"
	"unicode"
)

const (
	channelIdsKey          = "channelIds"
	frequencyKey           = "frequency"
	FingerQuoterPluginName = "fingerQuoter"
)

// FingerQuoter holds the plugin data for the finger quoter plugin
type FingerQuoter struct {
	slackscot.Plugin
}

// NewFingerQuoter creates a new instance of the plugin
func NewFingerQuoter(config *config.PluginConfig) (p *FingerQuoter, err error) {
	var channels []string

	channelValue := config.GetString(channelIdsKey)
	channels = strings.Split(channelValue, ",")

	if ok := config.IsSet(frequencyKey); !ok {
		return nil, fmt.Errorf("Missing %s config key: %s", FingerQuoterPluginName, frequencyKey)
	}

	frequency := config.GetInt(frequencyKey)

	return &FingerQuoter{slackscot.Plugin{Name: "fingerQuoter", Commands: nil, HearActions: []slackscot.ActionDefinition{{
		Hidden: true,
		// Match based on the frequency probability and whether or not the channel is whitelisted
		Match: func(t string, m *slack.Msg) bool {
			if !isChannelWhiteListed(m.Channel, channels) {
				return false
			}

			f, err := strconv.ParseFloat(m.Timestamp, 64)
			if err != nil {
				slackscot.Debugf("[%s] Skipping message [%s] because of error converting timestamp to float: %v\n", FingerQuoterPluginName, m, err)
			} else {
				// Make the random generator use a seed based on the message id so that we preserve the same matches when messages get updated
				randomGen := rand.New(rand.NewSource(int64(f)))

				// Determine if we're going to react this time or not
				return randomGen.Int31n(int32(frequency)) == 0
			}
			return false
		},
		Usage:       "just speak",
		Description: "finger quoter listens to what people say and (sometimes) finger quotes a word",
		Answer: func(m *slack.Msg) string {
			candidates := splitInputIntoWordsLongerThan(m.Text, 4)

			if len(candidates) > 0 {

				f, err := strconv.ParseFloat(m.Timestamp, 64)
				if err != nil {
					slackscot.Debugf("[%s] Skipping message [%s] because of error converting timestamp to float: %v\n", FingerQuoterPluginName, m, err)
				} else {
					// Make the random generator use a seed based on the message id so that we preserve the same matches when messages get updated
					randomGen := rand.New(rand.NewSource(int64(f)))

					i := randomGen.Int31n(int32(len(candidates)))
					return fmt.Sprintf("\"%s\"", candidates[i])
				}
			}

			// Not this time, skip
			return ""
		},
	}}}}, nil
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
