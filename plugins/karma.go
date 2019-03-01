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
	karmaStorer store.StringStorer
}

const (
	// KarmaPluginName holds identifying name for the karma plugin
	KarmaPluginName = "karma"
)

var karmaRegex = regexp.MustCompile("(?:\\A|\\W)<?(@?[\\w']+-?[\\w']+)>?(\\+{2}|\\-{2}).*")
var topKarmaRegexp = regexp.MustCompile("(?i)(karma top)+ (\\d+).*")
var worstKarmaRegexp = regexp.MustCompile("(?i)(karma worst)+ (\\d+).*")

// NewKarma creates a new instance of the Karma plugin
func NewKarma(strStorer store.StringStorer) (karma *Karma) {
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
			Description: "Return the X top things ever",
			Answer:      k.answerKarmaTop,
		},
		{
			Hidden:      false,
			Match:       matchKarmaWorstReport,
			Usage:       "karma worst <howMany>",
			Description: "Return the X worst things ever",
			Answer:      k.answerKarmaWorst,
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

// recordKarma records a karma increase or decrease and answers with a message including
// the recorded word with its associated karma value
func (k *Karma) recordKarma(message *slackscot.IncomingMessage) *slackscot.Answer {
	match := karmaRegex.FindAllStringSubmatch(message.Text, -1)[0]

	var format string
	thing := match[1]
	rawValue, err := k.karmaStorer.GetString(thing)
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
	err = k.karmaStorer.PutString(thing, strconv.Itoa(karma))
	if err != nil {
		k.Logger.Printf("[%s] Error persisting karma: %v", KarmaPluginName, err)
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
	return k.answerKarmaRankList(topKarmaRegexp, message, "top", getTopThings)
}

func (k *Karma) answerKarmaWorst(message *slackscot.IncomingMessage) *slackscot.Answer {
	return k.answerKarmaRankList(worstKarmaRegexp, message, "worst", getWorstThings)
}

type extractRankedList func(rawData map[string]string, count int) (results pairList, err error)

func (k *Karma) answerKarmaRankList(regexp *regexp.Regexp, message *slackscot.IncomingMessage, rankingType string, getRankedItems extractRankedList) *slackscot.Answer {
	match := regexp.FindAllStringSubmatch(message.Text, -1)[0]

	rawCount := match[2]
	count, _ := strconv.Atoi(rawCount)

	values, err := k.karmaStorer.Scan()
	if err != nil {
		return &slackscot.Answer{Text: fmt.Sprintf("Sorry, I couldn't get the %s [%d] things for you. If you must know, thing happened: %v", rankingType, count, err)}
	}

	pairs, err := getRankedItems(values, count)
	if err != nil {
		return &slackscot.Answer{Text: fmt.Sprintf("Sorry, I couldn't get the %s [%d] things for you. If you must know, thing happened: %v", rankingType, count, err)}
	}
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("Here are the %s %d things: \n", rankingType, count))
	buffer.WriteString(k.formatList(pairs))
	return &slackscot.Answer{Text: buffer.String()}
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

func (p pairList) Len() int           { return len(p) }
func (p pairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p pairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func convertTopairs(wordFrequencies map[string]int) pairList {
	pl := make(pairList, len(wordFrequencies))
	i := 0
	for k, v := range wordFrequencies {
		pl[i] = pair{k, v}
		i++
	}

	return pl
}

func getTopThings(rawData map[string]string, count int) (results pairList, err error) {
	wordWithFrequencies, err := convertMapValues(rawData)
	if err != nil {
		return results, err
	}

	pl := convertTopairs(wordWithFrequencies)

	sort.Sort(sort.Reverse(pl))
	limit := count

	if len(pl) < count {
		limit = len(pl)
	}
	return pl[:limit], nil
}

func getWorstThings(rawData map[string]string, count int) (results pairList, err error) {
	wordWithFrequencies, err := convertMapValues(rawData)
	if err != nil {
		return results, err
	}

	pl := convertTopairs(wordWithFrequencies)

	sort.Sort(pl)

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
