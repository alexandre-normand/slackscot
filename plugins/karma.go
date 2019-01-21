package plugins

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/alexandre-normand/slackscot/v2"
	"github.com/alexandre-normand/slackscot/v2/config"
	"github.com/alexandre-normand/slackscot/v2/store"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// Karma holds the plugin data for the karma plugin
type Karma struct {
	slackscot.Plugin
	karmaStore *store.Store
}

const (
	// KarmaPluginName holds identifying name for the karma plugin
	KarmaPluginName = "karma"
)

var karmaRegex = regexp.MustCompile("\\s*(\\w+)(\\+\\+|\\-\\-).*")
var topKarmaRegexp = regexp.MustCompile("(?i)(karma top)+ (\\d+).*")
var worstKarmaRegexp = regexp.MustCompile("(?i)(karma worst)+ (\\d+).*")

// NewKarma creates a new instance of the Karma plugin
func NewKarma(v *viper.Viper) (karma *Karma, err error) {
	if !v.IsSet(config.StoragePathKey) {
		return nil, fmt.Errorf("Missing [%s] configuration key in the top value configuration", config.StoragePathKey)
	}

	storagePath := v.GetString(config.StoragePathKey)
	storage, err := store.New(KarmaPluginName, storagePath)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Opening [%s] db failed with path [%s]", KarmaPluginName, storagePath))
	}

	k := new(Karma)

	hearActions := []slackscot.ActionDefinition{
		{
			Hidden: false,
			Match: func(t string, m *slack.Msg) bool {
				matches := karmaRegex.FindStringSubmatch(t)
				return len(matches) > 0
			},
			Usage:       "thing++ or thing--",
			Description: "Keep track of karma",
			Answer:      k.recordKarma,
		}}

	commands := []slackscot.ActionDefinition{
		{
			Hidden: false,
			Match: func(t string, m *slack.Msg) bool {
				return strings.HasPrefix(t, "karma top")
			},
			Usage:       "karma top <howMany>",
			Description: "Return the X top things ever",
			Answer:      k.answerKarmaTop,
		},
		{
			Hidden: false,
			Match: func(t string, m *slack.Msg) bool {
				return strings.HasPrefix(t, "karma worst")
			},
			Usage:       "karma worst <howMany>",
			Description: "Return the X worst things ever",
			Answer:      k.answerKarmaWorst,
		},
	}

	k.Plugin = slackscot.Plugin{Name: KarmaPluginName, Commands: commands, HearActions: hearActions}
	k.karmaStore = storage

	return k, nil
}

func (k *Karma) recordKarma(message *slack.Msg) string {
	match := karmaRegex.FindAllStringSubmatch(message.Text, -1)[0]

	var format string
	thing := match[1]
	rawValue, err := k.karmaStore.Get(thing)
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
	err = k.karmaStore.Put(thing, strconv.Itoa(karma))
	if err != nil {
		k.Logger.Printf("[%s] Error persisting karma: %v", KarmaPluginName, err)
	}
	return fmt.Sprintf(format, thing, thing, karma)

}

func (k *Karma) answerKarmaTop(message *slack.Msg) string {
	return k.answerKarmaRankList(topKarmaRegexp, message, "top", getTopThings)
}

func (k *Karma) answerKarmaWorst(message *slack.Msg) string {
	return k.answerKarmaRankList(worstKarmaRegexp, message, "worst", getWorstThings)
}

type extractRankedList func(rawData map[string]string, count int) (results pairList, err error)

func (k *Karma) answerKarmaRankList(regexp *regexp.Regexp, message *slack.Msg, rankingType string, getRankedItems extractRankedList) string {
	match := regexp.FindAllStringSubmatch(message.Text, -1)[0]

	rawCount := match[2]
	count, _ := strconv.Atoi(rawCount)

	values, err := k.karmaStore.Scan()
	if err != nil {
		return fmt.Sprintf("Sorry, I couldn't get the %s [%d] things for you. If you must know, thing happened: %v", rankingType, count, err)
	}

	pairs, err := getRankedItems(values, count)
	if err != nil {
		return fmt.Sprintf("Sorry, I couldn't get the %s [%d] things for you. If you must know, thing happened: %v", rankingType, count, err)
	}
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("Here are the %s %d things: \n", rankingType, count))
	buffer.WriteString(formatList(pairs))
	return buffer.String()
}

func formatList(pl pairList) string {
	var b bytes.Buffer
	b.WriteString("```")
	w := new(tabwriter.Writer)
	bufw := bufio.NewWriter(&b)
	w.Init(bufw, 5, 0, 1, ' ', 0)
	for _, pair := range pl {
		fmt.Fprintf(w, "%d\t%s\n", pair.Value, pair.Key)
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

// Close closes the plugin and its underlying database
func (k *Karma) Close() {
	if k.karmaStore != nil {
		k.karmaStore.Close()
	}
}
