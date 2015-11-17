package brain

import (
	"errors"
	"fmt"
	"github.com/alexandre-normand/slack"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
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

func (fingerQuoter FingerQuoter) Init(config config.Configuration) (commands []slackscot.Action, listeners []slackscot.Action, err error) {
	fingerQuoterRegex := regexp.MustCompile("(?i)([a-zA-Z\\-]{5,16})+")

	var channels []string
	frequency := 0

	if extensionConfig, ok := config.Extentions[fingerQuoter.String()]; !ok {
		return nil, nil, errors.New(fmt.Sprintf("Missing extention config for %s", fingerQuoter.String()))
	} else {
		if channelValue, ok := extensionConfig[CHANNELS]; !ok {
			return nil, nil, errors.New(fmt.Sprintf("Missing %s config key: %s", fingerQuoter.String(), CHANNELS))
		} else {
			channels = strings.Split(channelValue, ",")
		}

		if frequencyValue, ok := extensionConfig[FREQUENCY]; !ok {
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

	listeners = append(listeners, slackscot.Action{
		Hidden:      true,
		Regex:       fingerQuoterRegex,
		Usage:       "just speak",
		Description: "finger quoter listens to what people say and (sometimes) finger quotes a word",
		Answerer: func(message *slack.Message) string {
			if isChannelWhiteListed(message.ChannelId, channels) {
				words := fingerQuoterRegex.FindAllString(message.Text, -1)
				log.Printf("All words are: %v", words)

				// Determine if we're going to react this time or not
				if randomGen.Int31n(int32(frequency)) == 0 {
					// That's it, let's pick a word and finger-quote it
					i := randomGen.Int31n(int32(len(words)))
					return fmt.Sprintf("\"%s\"", words[i])
				}
			} else {
				log.Printf("Channel [%s] is not whitelisted.", message.ChannelId)
			}
			// Not this time, skip
			return ""
		},
	})

	return commands, listeners, nil
}

func isChannelWhiteListed(channelId string, whitelist []string) bool {
	for _, c := range whitelist {
		if c == channelId {
			return true
		}
	}

	return false
}

func (fingerQuoter FingerQuoter) Close() {

}
