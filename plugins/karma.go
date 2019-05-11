package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/actions"
	"github.com/alexandre-normand/slackscot/plugin"
	"github.com/alexandre-normand/slackscot/store"
	"github.com/nlopes/slack"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Karma holds the plugin data for the karma plugin
type Karma struct {
	*slackscot.Plugin
	karmaStorer store.GlobalSiloStringStorer
}

const (
	// KarmaPluginName holds identifying name for the karma plugin
	KarmaPluginName  = "karma"
	defaultItemCount = 5
)

var karmaRegex = regexp.MustCompile("(?:\\A|\\W)(?:(?:<(@[\\w']+)>\\s?)|([\\w']+-?[\\w']+))(\\+{2,6}|\\-{2,6}).*")

// Ranker represents attributes and behavior to process a ranking list
type ranker struct {
	name             string
	regexp           *regexp.Regexp
	bannerText       string
	bannerImgLink    string
	bannerImgAltText string
	scanner          karmaScanner
	sorter           karmaSorter
}

var globalTopRanker ranker
var topRanker ranker
var globalWorstRanker ranker
var worstRanker ranker

func init() {
	globalTopRanker = ranker{name: "global top",
		regexp:     regexp.MustCompile("(?i)\\A(global top)+(?:\\s+(\\d*))*\\z"),
		bannerText: ":leaves::leaves::leaves::trophy: *Global Top* :trophy::leaves::leaves::leaves:",
		scanner:    scanGlobalKarma,
		sorter:     sortTop}

	topRanker = ranker{name: "top",
		regexp:     regexp.MustCompile("(?i)\\A(top)+(?:\\s+(\\d*))*\\z"),
		bannerText: ":leaves::leaves::leaves::trophy: *Top* :trophy::leaves::leaves::leaves:",
		scanner:    scanChannelKarma,
		sorter:     sortTop}

	globalWorstRanker = ranker{name: "global worst",
		regexp:     regexp.MustCompile("(?i)\\A(global worst)+(?:\\s+(\\d*))*\\z"),
		bannerText: ":fallen_leaf::fallen_leaf::fallen_leaf::space_invader: *Global Worst* :space_invader::fallen_leaf::fallen_leaf::fallen_leaf:",
		scanner:    scanGlobalKarma,
		sorter:     sortWorst}

	worstRanker = ranker{name: "worst",
		regexp:     regexp.MustCompile("(?i)\\A(worst)+(?:\\s+(\\d*))*\\z"),
		bannerText: ":fallen_leaf::fallen_leaf::fallen_leaf::space_invader: *Worst* :space_invader::fallen_leaf::fallen_leaf::fallen_leaf:",
		scanner:    scanChannelKarma,
		sorter:     sortWorst}
}

// NewKarma creates a new instance of the Karma plugin
func NewKarma(storer store.GlobalSiloStringStorer) (karma *slackscot.Plugin) {
	k := new(Karma)

	k.Plugin = plugin.New(KarmaPluginName).
		WithCommandNamespacing().
		WithCommand(actions.NewCommand().
			WithMatcher(matchKarmaTopReport).
			WithUsage("top [count]").
			WithDescriptionf("Return the top things ever recorded in this channel (default of %d items)", defaultItemCount).
			WithAnswerer(k.answerKarmaTop).
			Build()).
		WithCommand(actions.NewCommand().
			WithMatcher(matchKarmaWorstReport).
			WithUsage("worst [count]").
			WithDescriptionf("Return the worst things ever recorded in this channel (default of %d items)", defaultItemCount).
			WithAnswerer(k.answerKarmaWorst).
			Build()).
		WithCommand(actions.NewCommand().
			WithMatcher(matchGlobalKarmaTopReport).
			WithUsage("global top [count]").
			WithDescriptionf("Return the top things ever over all channels (default of %d items)", defaultItemCount).
			WithAnswerer(k.answerGlobalKarmaTop).
			Build()).
		WithCommand(actions.NewCommand().
			WithMatcher(matchGlobalKarmaWorstReport).
			WithUsage("global worst [count]").
			WithDescriptionf("Return the worst things ever over all channels (default of %d items)", defaultItemCount).
			WithAnswerer(k.answerGlobalKarmaWorst).
			Build()).
		WithCommand(actions.NewCommand().
			Hidden().
			WithMatcher(matchKarmaReset).
			WithUsage("reset").
			WithDescription("Resets all recorded karma for the current channel").
			WithAnswerer(k.clearChannelKarma).
			Build()).
		WithHearAction(actions.NewCommand().
			WithMatcher(matchKarmaRecord).
			WithUsage("thing++ or thing--").
			WithDescription("Keep track of karma. Increments larger than `1` (up to `5`) can be achieved with extra `+` or `-` signs").
			WithAnswerer(k.recordKarma).
			Build()).
		Build()

	k.karmaStorer = storer

	return k.Plugin
}

// matchKarmaRecord returns true if the message matches karma++ or karma-- (karma being any word)
func matchKarmaRecord(m *slackscot.IncomingMessage) bool {
	matches := karmaRegex.FindStringSubmatch(m.NormalizedText)
	return len(matches) > 0
}

// matchKarmaTopReport returns true if the message matches a request for top karma with
// a message such as "top <count>"
func matchKarmaTopReport(m *slackscot.IncomingMessage) bool {
	return topRanker.regexp.MatchString(m.NormalizedText)
}

// matchKarmaWorstReport returns true if the message matches a request for the worst karma with
// a message such as "worst <count>"
func matchKarmaWorstReport(m *slackscot.IncomingMessage) bool {
	return worstRanker.regexp.MatchString(m.NormalizedText)
}

// matchGlobalKarmaTopReport returns true if the message matches a request for top global karma with
// a message such as "global top <count>"
func matchGlobalKarmaTopReport(m *slackscot.IncomingMessage) bool {
	return globalTopRanker.regexp.MatchString(m.NormalizedText)
}

// matchGlobalKarmaWorstReport returns true if the message matches a request for the worst global karma with
// a message such as "global worst <count>"
func matchGlobalKarmaWorstReport(m *slackscot.IncomingMessage) bool {
	return globalWorstRanker.regexp.MatchString(m.NormalizedText)
}

// matchKarmaReset returns true if the message matches a request for resetting karma with a
// message such as "reset"
func matchKarmaReset(m *slackscot.IncomingMessage) bool {
	return strings.HasPrefix(m.NormalizedText, "reset")
}

// recordKarma records a karma increase or decrease and answers with a message including
// the recorded word with its associated karma value
func (k *Karma) recordKarma(message *slackscot.IncomingMessage) *slackscot.Answer {
	match := karmaRegex.FindAllStringSubmatch(message.Text, -1)[0]

	// Depending on if it's a user id or a "normal" thing, the matching group is different so we
	// check both (only one can ever match)
	thing := match[1]
	if len(thing) > 0 {
		// Prevent a user from attributing karma to self
		if strings.TrimPrefix(thing, "@") == message.User {
			return &slackscot.Answer{Text: "*Attributing yourself karma is frown upon* :face_with_raised_eyebrow:", Options: []slackscot.AnswerOption{slackscot.AnswerEphemeral(message.User)}}
		}
	} else {
		thing = match[2]
	}

	rawValue, err := k.karmaStorer.GetSiloString(message.Channel, thing)
	if err != nil {
		rawValue = "0"
	}
	karma, err := strconv.Atoi(rawValue)
	if err != nil {
		k.Logger.Printf("[%s] Error parsing current karma value [%s], something's wrong and resetting to 0: %v", KarmaPluginName, rawValue, err)
		karma = 0
	}

	answerText := ""
	renderedThing := k.renderThing(thing)

	if strings.HasPrefix(match[3], "+") {
		incrementSymbols := strings.TrimPrefix(match[3], "+")
		increment := len(incrementSymbols)
		karma = karma + increment

		if increment == 1 {
			answerText = fmt.Sprintf("`%s` just gained a level (`%s`: %d)", renderedThing, renderedThing, karma)
		} else {
			answerText = fmt.Sprintf("`%s` just gained %d levels (`%s`: %d)", renderedThing, increment, renderedThing, karma)
		}

	} else {
		decrementSymbols := strings.TrimPrefix(match[3], "-")
		decrement := len(decrementSymbols)
		karma = karma - decrement

		if decrement == 1 {
			answerText = fmt.Sprintf("`%s` just lost a life (`%s`: %d)", renderedThing, renderedThing, karma)
		} else {
			answerText = fmt.Sprintf("`%s` just lost %d lives (`%s`: %d)", renderedThing, decrement, renderedThing, karma)
		}
	}

	// Store new value
	err = k.karmaStorer.PutSiloString(message.Channel, thing, strconv.Itoa(karma))
	if err != nil {
		k.Logger.Printf("[%s] Error persisting karma: %v", KarmaPluginName, err)
		return nil
	}

	return &slackscot.Answer{Text: answerText}
}

// renderThing renders the thing value. In most cases, it should just return the value
// untouched but if it starts with '@', it tries to find the user info matching the value
// and returns that instead (if found a match)
func (k *Karma) renderThing(thing string) (renderedThing string) {
	if strings.HasPrefix(thing, "@") {
		u, _ := k.UserInfoFinder.GetUserInfo(strings.TrimPrefix(thing, "@"))

		if u != nil {
			return u.RealName
		}
	}

	return thing
}

// answerKarmaTop returns an answer with the top list of karma entries for the channel the message is received on
func (k *Karma) answerKarmaTop(m *slackscot.IncomingMessage) *slackscot.Answer {
	return k.answerKarmaRankList(m, topRanker)
}

// answerKarmaTop returns an answer with the list of worst karma entries for the channel the message is received on
func (k *Karma) answerKarmaWorst(m *slackscot.IncomingMessage) *slackscot.Answer {
	return k.answerKarmaRankList(m, worstRanker)
}

// answerKarmaTop returns an answer with the top list of karma entries for all channels
func (k *Karma) answerGlobalKarmaTop(m *slackscot.IncomingMessage) *slackscot.Answer {
	return k.answerKarmaRankList(m, globalTopRanker)
}

// answerKarmaTop returns an answer with the list of worst karma entries for all channels
func (k *Karma) answerGlobalKarmaWorst(m *slackscot.IncomingMessage) *slackscot.Answer {
	return k.answerKarmaRankList(m, globalWorstRanker)
}

// clearChannelKarma processes a request to clear karma in a channel (the message's channel is used to tell which one)
func (k *Karma) clearChannelKarma(m *slackscot.IncomingMessage) *slackscot.Answer {
	entries, err := k.karmaStorer.ScanSilo(m.Channel)
	if err != nil {
		return &slackscot.Answer{Text: fmt.Sprintf("Sorry, I couldn't get delete karma for channel [%s] for you. If you must know, this happened: %s", m.Channel, err.Error())}
	}

	for thing, _ := range entries {
		err = k.karmaStorer.DeleteSiloString(m.Channel, thing)
	}

	if err != nil {
		return &slackscot.Answer{Text: fmt.Sprintf("Sorry, I couldn't get delete karma for channel [%s] for you. If you must know, this happened: %s", m.Channel, err.Error())}
	}

	return &slackscot.Answer{Text: "karma all cleared :white_check_mark::boom:"}
}

// karmaSorter is a function sorting pairList of karma entries. Used to plug in top/worst sorting
type karmaSorter func(pl pairList)

// sortWorst sorts karma from the lowest value to the highest
func sortWorst(pl pairList) {
	sort.Sort(pl)
}

// sortWorst sorts karma from the highest to lowest
func sortTop(pl pairList) {
	sort.Sort(sort.Reverse(pl))
}

// karmaScanner is a function that returns karma entries for a given channel. It is used
// to plug in different behaviors like channel scanning and global scanning
type karmaScanner func(karmaStorer store.GlobalSiloStringStorer, channelID string) (entries map[string]string, err error)

// scanChannelKarma scans the silo for the given channel id and returns only the entries for that
// channel
func scanChannelKarma(karmaStorer store.GlobalSiloStringStorer, channelID string) (entries map[string]string, err error) {
	return karmaStorer.ScanSilo(channelID)
}

// scanGlobalKarma invokes a GlobalScan and merges karma over all channels. If there's
// an error, a nil map is returned along with that error
func scanGlobalKarma(karmaStorer store.GlobalSiloStringStorer, channelID string) (entries map[string]string, err error) {
	entriesByChannel, err := karmaStorer.GlobalScan()
	if err != nil {
		return nil, err
	}

	entries = make(map[string]string)
	for _, chEntries := range entriesByChannel {
		for thing, val := range chEntries {
			if _, ok := entries[thing]; !ok {
				entries[thing] = val
			} else {
				entries[thing], err = mergeKarma(entries[thing], val)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return entries, nil
}

// mergeKarma merges two values assumed to be strings holding integers and
// returns the sum as a string
func mergeKarma(v1 string, v2 string) (merged string, err error) {
	val1, err := strconv.Atoi(v1)
	if err != nil {
		return "", err
	}

	val2, err := strconv.Atoi(v2)
	if err != nil {
		return "", err
	}

	return strconv.Itoa(val1 + val2), nil
}

// answerKarmaRankList returns an answer for a ranked list request according to the behavior and attributes of the given ranker
func (k *Karma) answerKarmaRankList(m *slackscot.IncomingMessage, ranker ranker) *slackscot.Answer {
	match := ranker.regexp.FindAllStringSubmatch(m.NormalizedText, -1)[0]

	count := defaultItemCount
	rawCount := match[2]
	if len(rawCount) > 0 {
		count, _ = strconv.Atoi(rawCount)
	}

	values, err := ranker.scanner(k.karmaStorer, m.Channel)
	if err != nil {
		return &slackscot.Answer{Text: fmt.Sprintf("Sorry, I couldn't get the %s [%d] things for you. If you must know, this happened: %v", ranker.name, count, err)}
	}

	pairs, err := getRankedList(values, count, ranker.sorter)
	if err != nil {
		return &slackscot.Answer{Text: fmt.Sprintf("Sorry, I couldn't get the %s [%d] things for you. If you must know, this happened: %v", ranker.name, count, err)}
	}

	if len(pairs) > 0 {
		blocks := make([]slack.Block, 0)

		blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", ranker.bannerText, false, false), nil, nil))
		blocks = append(blocks, k.formatList(pairs)...)

		return &slackscot.Answer{Text: "", ContentBlocks: blocks}
	}

	return &slackscot.Answer{Text: "Sorry, no recorded karma found :disappointed:"}
}

// formatList formats a list of ranked items using the rankRenderer to render the rank icons and returns the resulting block kit blocks
func (k *Karma) formatList(pl pairList) (blocks []slack.Block) {
	blocks = make([]slack.Block, 0)

	rank := 1
	for _, pair := range pl {
		blocks = append(blocks, formatRankedElement(pair, rank))
		rank = rank + 1
	}

	return blocks
}

// formatRankedElement formats one ranked element in a list. It adds 3 blocks: one for the rank (icon),
// one for the ranked "thing" and one for its karma value. The 3 block objects are then wrapped in a context block
func formatRankedElement(p pair, rank int) (block slack.Block) {
	return *slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("• %s `%d`", renderThingName(p.Key), p.Value), false, false), nil, nil)
}

// renderThingName renders a karma item by formatting a user id with the required symbols such that it looks
// like <@userId>. For things that aren't user ids, the value is returned as-is
func renderThingName(thing string) (render string) {
	if strings.HasPrefix(thing, "@") {
		return "<" + thing + ">"
	}

	return thing
}

// pair holds a key (thing name) and its count
type pair struct {
	Key   string
	Value int
}

// pairList adapted from Andrew Gerrand for a similar problem: https://groups.google.com/forum/#!topic/golang-nuts/FT7cjmcL7gw
type pairList []pair

func (p pairList) Len() int { return len(p) }

func (p pairList) Less(i, j int) bool {
	return p[i].Value < p[j].Value || (p[i].Value == p[j].Value && strings.Compare(p[i].Key, p[j].Key) > 0)
}

func (p pairList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func convertToPairs(wordFrequencies map[string]int) pairList {
	pl := make(pairList, len(wordFrequencies))
	i := 0
	for k, v := range wordFrequencies {
		pl[i] = pair{k, v}
		i++
	}

	return pl
}

func getRankedList(rawData map[string]string, count int, sort karmaSorter) (results pairList, err error) {
	wordWithFrequencies, err := convertMapValues(rawData)
	if err != nil {
		return results, err
	}

	pl := convertToPairs(wordWithFrequencies)

	sort(pl)

	limit := count

	if len(pl) < count {
		limit = len(pl)
	}
	return pl[:limit], nil
}

func convertMapValues(rawData map[string]string) (result map[string]int, err error) {
	result = map[string]int{}

	for k, v := range rawData {
		result[k], err = strconv.Atoi(v)
		if err != nil {
			return result, err
		}
	}

	return result, nil
}
