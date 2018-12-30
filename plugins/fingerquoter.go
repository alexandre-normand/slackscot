package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/nlopes/slack"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	channelIdsKey          = "channelIds"
	frequencyKey           = "frequency"
	fingerQuoterPluginName = "fingerQuoter"
)

type FingerQuoter struct {
	slackscot.Plugin
}

func NewFingerQuoter(config config.Configuration) (p *FingerQuoter, err error) {
	fingerQuoterRegex := regexp.MustCompile("(?i)([a-zA-Z\\-]{5,16})+")

	var channels []string
	frequency := 0

	if pluginConfig, ok := config.Plugins[fingerQuoterPluginName]; !ok {
		return nil, fmt.Errorf("Missing plugin config for %s", fingerQuoterPluginName)
	} else {
		if channelValue, ok := pluginConfig[channelIdsKey]; ok {
			channels = strings.Split(channelValue, ",")
		}

		if frequencyValue, ok := pluginConfig[frequencyKey]; !ok {
			return nil, fmt.Errorf("Missing %s config key: %s", fingerQuoterPluginName, frequencyKey)
		} else {
			frequency, err = strconv.Atoi(frequencyValue)
			if err != nil {
				return nil, err
			}
		}
	}

	return &FingerQuoter{slackscot.Plugin{Name: "fingerQuoter", Commands: nil, HearActions: []slackscot.ActionDefinition{slackscot.ActionDefinition{
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
				slackscot.Debugf(config, "Channel [%s] is not whitelisted.", m.Channel)
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
