package plugins

import (
	"errors"
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	CHANNELS  = "channelIds"
	FREQUENCY = "frequency"
)

type FingerQuoter struct {
}

func NewFingerQuoter() *FingerQuoter {
	return &FingerQuoter{}
}

func (fingerQuoter FingerQuoter) String() string {
	return "fingerQuoter"
}

func (fingerQuoter FingerQuoter) Init(config config.Configuration) (commands []slackscot.ActionDefinition, listeners []slackscot.ActionDefinition, err error) {
	fingerQuoterRegex := regexp.MustCompile("(?i)([a-zA-Z\\-]{5,16})+")

	var channels []string
	frequency := 0

	if pluginConfig, ok := config.Plugins[fingerQuoter.String()]; !ok {
		return nil, nil, errors.New(fmt.Sprintf("Missing extention config for %s", fingerQuoter.String()))
	} else {
		if channelValue, ok := pluginConfig[CHANNELS]; ok {
			channels = strings.Split(channelValue, ",")
		}

		if frequencyValue, ok := pluginConfig[FREQUENCY]; !ok {
			return nil, nil, errors.New(fmt.Sprintf("Missing %s config key: %s", fingerQuoter.String(), FREQUENCY))
		} else {
			frequency, err = strconv.Atoi(frequencyValue)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	randomGen := rand.New(rand.NewSource(time.Now().UnixNano()))
	log.Printf("%s loaded with frequency [%d] and whitelist [%s]", fingerQuoter, frequency, channels)

	listeners = append(listeners, slackscot.ActionDefinition{
		Hidden:      true,
		Regex:       fingerQuoterRegex,
		Usage:       "just speak",
		Description: "finger quoter listens to what people say and (sometimes) finger quotes a word",
		Answerer: func(me *slackscot.IncomingMessageEvent) string {
			if isChannelWhiteListed(me.Channel, channels) {
				words := strings.FieldsFunc(me.Text, func(c rune) bool {
					return !unicode.IsLetter(c) && c != '-'
				})

				candidates := filterWordsLongerThan(words, 5)

				// Determine if we're going to react this time or not
				if randomGen.Int31n(int32(frequency)) == 0 {
					// That's it, let's pick a word and finger-quote it
					i := randomGen.Int31n(int32(len(candidates)))
					return fmt.Sprintf("\"%s\"", candidates[i])
				}
			} else if config.Debug {
				log.Printf("Channel [%s] is not whitelisted.", me.Channel)
			}
			// Not this time, skip
			return ""
		},
	})

	return commands, listeners, nil
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

func (fingerQuoter FingerQuoter) Close() {

}
