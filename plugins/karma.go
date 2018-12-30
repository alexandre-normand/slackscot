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
	"log"
	"regexp"
	"sort"
	"strconv"
	"text/tabwriter"
)

type Karma struct {
	slackscot.Plugin
	karmaStore *store.Store
}

const (
	karmaPluginName = "karma"
)

func NewKarma(c config.Configuration) (karma *Karma, err error) {
	storage, err := store.NewStore(karmaPluginName, c.StoragePath)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Opening [%s] db failed with path [%s]", karmaPluginName, c.StoragePath))
	}

	//	v, err := storage.Get("test")
	log.Printf("Initialized storage successfully: %v", storage)
	karmaRegex := regexp.MustCompile("\\s*(\\w+)(\\+\\+|\\-\\-).*")

	hearActions := []slackscot.ActionDefinition{
		slackscot.ActionDefinition{
			Hidden:      false,
			Regex:       karmaRegex,
			Usage:       "thing++ or thing--",
			Description: "Keep track of karma",
			Answerer: func(message *slack.Msg) string {
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
		slackscot.ActionDefinition{
			Hidden:      false,
			Regex:       topKarmaRegexp,
			Usage:       "karma top <howMany>",
			Description: "Return the X top things ever",
			Answerer: func(message *slack.Msg) string {
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
		slackscot.ActionDefinition{
			Hidden:      false,
			Regex:       worstKarmaRegexp,
			Usage:       "karma worst <howMany>",
			Description: "Return the X worst things ever",
			Answerer: func(message *slack.Msg) string {
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

	karmaPlugin := Karma{Plugin: slackscot.Plugin{Name: karmaPluginName, Commands: commands, HearActions: hearActions}, karmaStore: storage}
	return &karmaPlugin, nil
}

func (karma Karma) Init(config config.Configuration) (commands []slackscot.ActionDefinition, listeners []slackscot.ActionDefinition, err error) {

	return commands, listeners, nil
}

func formatList(pl PairList) string {
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

type Pair struct {
	Key   string
	Value int
}

// PairList adapted from Andrew Gerrand for a similar problem: https://groups.google.com/forum/#!topic/golang-nuts/FT7cjmcL7gw
type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func convertToPairs(wordFrequencies map[string]int) PairList {
	pl := make(PairList, len(wordFrequencies))
	i := 0
	for k, v := range wordFrequencies {
		pl[i] = Pair{k, v}
		i++
	}

	return pl
}

func getTopThings(rawData map[string]string, count int) (results PairList, err error) {
	wordWithFrequencies, err := convertMapValues(rawData)
	if err != nil {
		return results, err
	}

	pl := convertToPairs(wordWithFrequencies)

	sort.Sort(sort.Reverse(pl))
	limit := count

	if len(pl) < count {
		limit = len(pl)
	}
	return pl[:limit], nil
}

func getWorstThings(rawData map[string]string, count int) (results PairList, err error) {
	wordWithFrequencies, err := convertMapValues(rawData)
	if err != nil {
		return results, err
	}

	pl := convertToPairs(wordWithFrequencies)

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

func (karma Karma) Close() {
	if karma.karmaStore != nil {
		karma.karmaStore.Close()
	}
}
