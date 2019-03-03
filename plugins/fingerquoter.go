package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/actions"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/plugin"
	"math/rand"
	"regexp"
	"strconv"
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
	*slackscot.Plugin
	channels        []string
	ignoredChannels []string
	frequency       int
}

// Regular expressions to find candidate words. They must be at least 5 characters long
// and can include any word character (include hyphen and underscore)
var candidateWordsStarting = regexp.MustCompile("(?:^|\\s)([\\w-]{5,})")
var candidateWordsEnding = regexp.MustCompile("([\\w-]{5,})(?:$|\\s)")

// NewFingerQuoter creates a new instance of the plugin
func NewFingerQuoter(config *config.PluginConfig) (p *slackscot.Plugin, err error) {
	if ok := config.IsSet(frequencyKey); !ok {
		return nil, fmt.Errorf("Missing %s config key: %s", FingerQuoterPluginName, frequencyKey)
	}

	f := new(FingerQuoter)
	f.channels = config.GetStringSlice(channelIDsKey)
	f.ignoredChannels = config.GetStringSlice(ignoredChannelIDsKey)
	f.frequency = config.GetInt(frequencyKey)

	f.Plugin = plugin.New(FingerQuoterPluginName).
		WithHearAction(actions.New().
			Hidden().
			WithMatcher(f.trigger).
			WithUsage("just converse").
			WithDescription("finger quoter listens to what people say and (sometimes) finger quotes a word").
			WithAnswerer(f.fingerQuoteMsg).
			Build()).
		Build()

	return f.Plugin, err
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
	candidates := findCandidateWords(m.NormalizedText)

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

// findCandidateWords looks at an input string and finds acceptable candidates for finger quoting
func findCandidateWords(t string) (candidates []string) {
	matchesStarting := candidateWordsStarting.FindAllStringSubmatch(t, -1)
	matchesEnding := candidateWordsEnding.FindAllStringSubmatch(t, -1)
	candidatesStarting := getWordMatches(matchesStarting)
	candidatesEnding := getWordMatches(matchesEnding)

	return intersection(candidatesStarting, candidatesEnding)
}

// getWordMatches returns an array of matching words given a raw array of matches
func getWordMatches(m [][]string) (words []string) {
	for _, match := range m {
		candidate := match[1]
		words = append(words, candidate)
	}

	return words
}

// intersection returns the common elements present in both a and b
func intersection(a []string, b []string) (intersection []string) {
	m := make(map[string]bool)

	for _, item := range a {
		m[item] = true
	}

	for _, item := range b {
		if _, ok := m[item]; ok {
			intersection = append(intersection, item)
		}
	}

	return intersection
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
