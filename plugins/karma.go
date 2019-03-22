package plugins

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/store"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// Karma holds the plugin data for the karma plugin
type Karma struct {
	slackscot.Plugin
	karmaStorer store.GlobalSiloStringStorer
}

const (
	// KarmaPluginName holds identifying name for the karma plugin
	KarmaPluginName = "karma"
)

var karmaRegex = regexp.MustCompile("(?:\\A|\\W)<?(@?[\\w']+-?[\\w']+)>?\\s?(\\+{2}|\\-{2}).*")
var topKarmaRegexp = regexp.MustCompile("(?i)\\A(karma top)+ (\\d+).*")
var worstKarmaRegexp = regexp.MustCompile("(?i)\\A(karma worst)+ (\\d+).*")
var topGlobalKarmaRegexp = regexp.MustCompile("(?i)\\A(karma global top)+ (\\d+).*")
var worstGlobalKarmaRegexp = regexp.MustCompile("(?i)\\A(karma global worst)+ (\\d+).*")

// NewKarma creates a new instance of the Karma plugin
func NewKarma(strStorer store.GlobalSiloStringStorer) (karma *Karma) {
	k := new(Karma)

	hearActions := []slackscot.ActionDefinition{
		{
			Hidden:      false,
			Match:       matchKarmaRecord,
			Usage:       "thing++ or thing--",
			Description: "Keep track of karma",
			Answer:      k.recordKarma,
		}}

	commands := []slackscot.ActionDefinition{
		{
			Hidden:      false,
			Match:       matchKarmaTopReport,
			Usage:       "karma top <howMany>",
			Description: "Return the X top things ever recorded in this channel",
			Answer:      k.answerKarmaTop,
		},
		{
			Hidden:      false,
			Match:       matchKarmaWorstReport,
			Usage:       "karma worst <howMany>",
			Description: "Return the X worst things ever recorded in this channel",
			Answer:      k.answerKarmaWorst,
		},
		{
			Hidden:      false,
			Match:       matchGlobalKarmaTopReport,
			Usage:       "karma global top <howMany>",
			Description: "Return the X top things ever over all channels",
			Answer:      k.answerGlobalKarmaTop,
		},
		{
			Hidden:      false,
			Match:       matchGlobalKarmaWorstReport,
			Usage:       "karma global worst <howMany>",
			Description: "Return the X worst things ever over all channels",
			Answer:      k.answerGlobalKarmaWorst,
		},
	}

	k.Plugin = slackscot.Plugin{Name: KarmaPluginName, Commands: commands, HearActions: hearActions}
	k.karmaStorer = strStorer

	return k
}

// matchKarmaRecord returns true if the message matches karma++ or karma-- (karma being any word)
func matchKarmaRecord(m *slackscot.IncomingMessage) bool {
	matches := karmaRegex.FindStringSubmatch(m.NormalizedText)
	return len(matches) > 0
}

// matchKarmaTopReport returns true if the message matches a request for top karma with
// a message such as "karma top <count>""
func matchKarmaTopReport(m *slackscot.IncomingMessage) bool {
	return topKarmaRegexp.MatchString(m.NormalizedText)
}

// matchKarmaWorstReport returns true if the message matches a request for the worst karma with
// a message such as "karma worst <count>""
func matchKarmaWorstReport(m *slackscot.IncomingMessage) bool {
	return worstKarmaRegexp.MatchString(m.NormalizedText)
}

// matchGlobalKarmaTopReport returns true if the message matches a request for top global karma with
// a message such as "global karma top <count>""
func matchGlobalKarmaTopReport(m *slackscot.IncomingMessage) bool {
	return topGlobalKarmaRegexp.MatchString(m.NormalizedText)
}

// matchGlobalKarmaWorstReport returns true if the message matches a request for the worst global karma with
// a message such as "global karma worst <count>""
func matchGlobalKarmaWorstReport(m *slackscot.IncomingMessage) bool {
	return worstGlobalKarmaRegexp.MatchString(m.NormalizedText)
}

// recordKarma records a karma increase or decrease and answers with a message including
// the recorded word with its associated karma value
func (k *Karma) recordKarma(message *slackscot.IncomingMessage) *slackscot.Answer {
	match := karmaRegex.FindAllStringSubmatch(message.Text, -1)[0]

	var format string
	thing := match[1]
	rawValue, err := k.karmaStorer.GetSiloString(message.Channel, thing)
	if err != nil {
		rawValue = "0"
	}
	karma, err := strconv.Atoi(rawValue)
	if err != nil {
		k.Logger.Printf("[%s] Error parsing current karma value [%s], something's wrong and resetting to 0: %v", KarmaPluginName, rawValue, err)
		karma = 0
	}

	if match[2] == "++" {
		format = "`%s` just gained a level (`%s`: %d)"
		karma++
	} else {
		format = "`%s` just lost a life (`%s`: %d)"
		karma--
	}

	// Store new value
	err = k.karmaStorer.PutSiloString(message.Channel, thing, strconv.Itoa(karma))
	if err != nil {
		k.Logger.Printf("[%s] Error persisting karma: %v", KarmaPluginName, err)
		return nil
	}

	return &slackscot.Answer{Text: fmt.Sprintf(format, k.renderThing(thing), k.renderThing(thing), karma)}
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

func (k *Karma) answerKarmaTop(message *slackscot.IncomingMessage) *slackscot.Answer {
	return k.answerKarmaRankList(topKarmaRegexp, message, "top", k.scanChannelKarma, sortTop)
}

func (k *Karma) answerKarmaWorst(message *slackscot.IncomingMessage) *slackscot.Answer {
	return k.answerKarmaRankList(worstKarmaRegexp, message, "worst", k.scanChannelKarma, sortWorst)
}

func (k *Karma) answerGlobalKarmaTop(message *slackscot.IncomingMessage) *slackscot.Answer {
	return k.answerKarmaRankList(topGlobalKarmaRegexp, message, "global top", k.scanGlobalKarma, sortTop)
}

func (k *Karma) answerGlobalKarmaWorst(message *slackscot.IncomingMessage) *slackscot.Answer {
	return k.answerKarmaRankList(worstGlobalKarmaRegexp, message, "global worst", k.scanGlobalKarma, sortWorst)
}

func sortWorst(pl pairList) {
	sort.Sort(pl)
}

func sortTop(pl pairList) {
	sort.Sort(sort.Reverse(pl))
}

type karmaSorter func(pl pairList)

func (k *Karma) scanChannelKarma(channelID string) (entries map[string]string, err error) {
	return k.karmaStorer.ScanSilo(channelID)
}

// scanGlobalKarma invokes a GlobalScan and merges karma over all channels. If there's
// an error, a nil map is returned along with that error
func (k *Karma) scanGlobalKarma(channelID string) (entries map[string]string, err error) {
	entriesByChannel, err := k.karmaStorer.GlobalScan()
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

type karmaScanner func(channel string) (entries map[string]string, err error)

func (k *Karma) answerKarmaRankList(regexp *regexp.Regexp, message *slackscot.IncomingMessage, rankingType string, scanner karmaScanner, sorter karmaSorter) *slackscot.Answer {
	match := regexp.FindAllStringSubmatch(message.NormalizedText, -1)[0]

	rawCount := match[2]
	count, _ := strconv.Atoi(rawCount)

	values, err := scanner(message.Channel)
	if err != nil {
		return &slackscot.Answer{Text: fmt.Sprintf("Sorry, I couldn't get the %s [%d] things for you. If you must know, this happened: %v", rankingType, count, err)}
	}

	pairs, err := getRankedList(values, count, sorter)
	if err != nil {
		return &slackscot.Answer{Text: fmt.Sprintf("Sorry, I couldn't get the %s [%d] things for you. If you must know, this happened: %v", rankingType, count, err)}
	}

	if len(pairs) > 0 {
		var buffer bytes.Buffer

		buffer.WriteString(fmt.Sprintf("Here are the %s %d things: \n", rankingType, min(len(pairs), count)))
		buffer.WriteString(k.formatList(pairs))

		return &slackscot.Answer{Text: buffer.String()}
	}

	return &slackscot.Answer{Text: "Sorry, no recorded karma found :disappointed:"}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (k *Karma) formatList(pl pairList) string {
	var b bytes.Buffer
	b.WriteString("```")
	w := new(tabwriter.Writer)
	bufw := bufio.NewWriter(&b)
	w.Init(bufw, 5, 0, 1, ' ', 0)
	for _, pair := range pl {
		fmt.Fprintf(w, "%d\t%s\n", pair.Value, k.renderThing(pair.Key))
	}
	fmt.Fprintf(w, "```\n")

	bufw.Flush()
	w.Flush()
	return b.String()
}

type pair struct {
	Key   string
	Value int
}

// pairList adapted from Andrew Gerrand for a similar problem: https://groups.google.com/forum/#!topic/golang-nuts/FT7cjmcL7gw
type pairList []pair

func (p pairList) Len() int { return len(p) }

func (p pairList) Less(i, j int) bool {
	return p[i].Value < p[j].Value || (p[i].Value == p[j].Value && strings.Compare(p[i].Key, p[j].Key) < 0)
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
