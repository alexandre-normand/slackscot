package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/nlopes/slack"
	"math/rand"
	"regexp"
	"strings"
	"time"
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
	fingerQuoterRegex := regexp.MustCompile("(?i)([a-zA-Z\\-]{5,16})+")

	var channels []string

	channelValue := config.GetString(channelIdsKey)
	channels = strings.Split(channelValue, ",")

	if ok := config.IsSet(frequencyKey); !ok {
		return nil, fmt.Errorf("Missing %s config key: %s", FingerQuoterPluginName, frequencyKey)
	}

	frequency := config.GetInt(frequencyKey)

	return &FingerQuoter{slackscot.Plugin{Name: "fingerQuoter", Commands: nil, HearActions: []slackscot.ActionDefinition{{
		Hidden:      true,
		Regex:       fingerQuoterRegex,
		Usage:       "just speak",
		Description: "finger quoter listens to what people say and (sometimes) finger quotes a word",
		Answerer: func(m *slack.Msg) string {
			if isChannelWhiteListed(m.Channel, channels) {
				words := strings.FieldsFunc(m.Text, func(c rune) bool {
					return !unicode.IsLetter(c) && c != '-'
				})

				candidates := filterWordsLongerThan(words, 4)

				if len(candidates) > 0 {
					randomGen := rand.New(rand.NewSource(time.Now().UnixNano()))
					// Determine if we're going to react this time or not
					if randomGen.Int31n(int32(frequency)) == 0 {
						// That's it, let's pick a word and finger-quote it
						i := randomGen.Int31n(int32(len(candidates)))
						return fmt.Sprintf("\"%s\"", candidates[i])
					}
				}
			} else {
				slackscot.Debugf("Channel [%s] is not whitelisted.", m.Channel)
			}
			// Not this time, skip
			return ""
		},
	}}}}, nil
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
