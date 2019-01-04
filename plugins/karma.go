package plugins

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/store"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"log"
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
	KarmaPluginName = "karma"
)

// NewKarma creates a new instance of the Karma plugin
func NewKarma(v *viper.Viper) (karma *Karma, err error) {
	storagePath := v.GetString(config.StoragePathKey)
	storage, err := store.NewStore(KarmaPluginName, storagePath)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Opening [%s] db failed with path [%s]", KarmaPluginName, storagePath))
	}

	slackscot.Debugf("Initialized storage successfully: %v", storage)

	karmaRegex := regexp.MustCompile("\\s*(\\w+)(\\+\\+|\\-\\-).*")

	hearActions := []slackscot.ActionDefinition{
		{
			Hidden: false,
			Match: func(t string, m *slack.Msg) bool {
				matches := karmaRegex.FindStringSubmatch(t)
				return len(matches) > 0
			},
			Usage:       "thing++ or thing--",
			Description: "Keep track of karma",
			Answer: func(message *slack.Msg) string {
				match := karmaRegex.FindAllStringSubmatch(message.Text, -1)[0]

				var format string
				thing := match[1]
				rawValue, err := storage.Get(thing)
				if err != nil {
					rawValue = "0"
				}
				k, err := strconv.Atoi(rawValue)
				if err != nil {
					log.Printf("Error parsing current karma value [%s], something's wrong and resetting to 0: %v", rawValue, err)
					k = 0
				}

				if match[2] == "++" {
					format = "`%s` just gained a level (`%s`: %d)"
					k++
				} else {
					format = "`%s` just lost a life (`%s`: %d)"
					k--
				}

				// Store new value
				err = karma.karmaStore.Put(thing, strconv.Itoa(k))
				if err != nil {
					log.Printf("Error persisting karma: %v", err)
				}
				return fmt.Sprintf(format, thing, thing, k)
			},
		}}

	topKarmaRegexp := regexp.MustCompile("(?i)(karma top)+ (\\d+).*")
	worstKarmaRegexp := regexp.MustCompile("(?i)(karma worst)+ (\\d+).*")

	commands := []slackscot.ActionDefinition{
		{
			Hidden: false,
			Match: func(t string, m *slack.Msg) bool {
				return strings.HasPrefix(t, "karma top")
			},
			Usage:       "karma top <howMany>",
			Description: "Return the X top things ever",
			Answer: func(message *slack.Msg) string {
				match := topKarmaRegexp.FindAllStringSubmatch(message.Text, -1)[0]
				log.Printf("Here are the matches: [%v]", match)

				rawCount := match[2]
				count, _ := strconv.Atoi(rawCount)

				values, err := storage.Scan()
				if err != nil {
					return fmt.Sprintf("Sorry, I couldn't get the top [%d] things for you. If you must know, thing happened: %v", count, err)
				}

				pairs, err := getTopThings(values, count)
				if err != nil {
					return fmt.Sprintf("Sorry, I couldn't get the top [%d] things for you. If you must know, thing happened: %v", count, err)
				}
				var buffer bytes.Buffer

				buffer.WriteString(fmt.Sprintf("Here are the top %d things: \n", count))
				buffer.WriteString(formatList(pairs))
				return buffer.String()
			},
		},
		{
			Hidden: false,
			Match: func(t string, m *slack.Msg) bool {
				return strings.HasPrefix(t, "karma worst")
			},
			Usage:       "karma worst <howMany>",
			Description: "Return the X worst things ever",
			Answer: func(message *slack.Msg) string {
				match := worstKarmaRegexp.FindAllStringSubmatch(message.Text, -1)[0]

				rawCount := match[2]
				count, _ := strconv.Atoi(rawCount)

				values, err := storage.Scan()
				if err != nil {
					return fmt.Sprintf("Sorry, I couldn't get the worst [%d] things for you. If you must know, thing happened: %v", count, err)
				}

				pairs, err := getWorstThings(values, count)
				if err != nil {
					return fmt.Sprintf("Sorry, I couldn't get the worst [%d] things for you. If you must know, thing happened: %v", count, err)
				}
				var buffer bytes.Buffer

				buffer.WriteString(fmt.Sprintf("Here are the %d worst things: \n", count))
				buffer.WriteString(formatList(pairs))
				return buffer.String()
			},
		},
	}

	karmaPlugin := Karma{Plugin: slackscot.Plugin{Name: KarmaPluginName, Commands: commands, HearActions: hearActions}, karmaStore: storage}
	return &karmaPlugin, nil
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
func (karma Karma) Close() {
	if karma.karmaStore != nil {
		karma.karmaStore.Close()
	}
}
