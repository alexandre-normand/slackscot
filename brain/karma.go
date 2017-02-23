package brain

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/alexandre-normand/slack"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/store"
	"log"
	"regexp"
	"sort"
	"strconv"
	"text/tabwriter"
)

type Karma struct {
	karmaStore *store.Store
}

func NewKarma() *Karma {
	return &Karma{karmaStore: nil}
}

func (karma Karma) String() string {
	return "karma"
}

func (karma Karma) Init(config config.Configuration) (commands []slackscot.Action, listeners []slackscot.Action, err error) {
	karma.karmaStore, err = store.NewStore("karma", config.StoragePath)
	if err != nil {
		return nil, nil, err
	}

	karmaRegex := regexp.MustCompile("\\s*(\\w+)(\\+\\+|\\-\\-).*")

	listeners = append(listeners, slackscot.Action{
		Hidden:      false,
		Regex:       karmaRegex,
		Usage:       "thing++ or thing--",
		Description: "Keep track of karma",
		Answerer: func(message *slack.Message) string {
			match := karmaRegex.FindAllStringSubmatch(message.Text, -1)[0]
			log.Printf("Here are the matches: [%v]", match)
			var format string
			thing := match[1]
			rawValue, err := karma.karmaStore.Get(thing)
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
	})

	topKarmaRegexp := regexp.MustCompile("(?i)(karma top)+ (\\d+).*")

	commands = append(commands, slackscot.Action{
		Hidden:      false,
		Regex:       topKarmaRegexp,
		Usage:       "karma top <howMany>",
		Description: "Return the X top things ever",
		Answerer: func(message *slack.Message) string {
			match := topKarmaRegexp.FindAllStringSubmatch(message.Text, -1)[0]
			log.Printf("Here are the matches: [%v]", match)

			rawCount := match[2]
			count, _ := strconv.Atoi(rawCount)

			values, err := karma.karmaStore.Scan()
			if err != nil {
				return fmt.Sprintf("Sorry, I couldn't get the top [%d] things for you. If you must know, thing happened: %v", count, err)
			}

			pairs, err := getTopThings(values, count)
			var buffer bytes.Buffer

			buffer.WriteString(fmt.Sprintf("Here are the top %d things: \n", count))
			buffer.WriteString(formatList(pairs))
			return buffer.String()
		},
	})

	worstKarmaRegexp := regexp.MustCompile("(?i)(karma worst)+ (\\d+).*")
	commands = append(commands, slackscot.Action{
		Hidden:      false,
		Regex:       worstKarmaRegexp,
		Usage:       "karma worst <howMany>",
		Description: "Return the X worst things ever",
		Answerer: func(message *slack.Message) string {
			match := worstKarmaRegexp.FindAllStringSubmatch(message.Text, -1)[0]
			log.Printf("Here are the matches: [%v]", match)

			rawCount := match[2]
			count, _ := strconv.Atoi(rawCount)

			values, err := karma.karmaStore.Scan()
			if err != nil {
				return fmt.Sprintf("Sorry, I couldn't get the worst [%d] things for you. If you must know, thing happened: %v", count, err)
			}

			pairs, err := getWorstThings(values, count)
			var buffer bytes.Buffer

			buffer.WriteString(fmt.Sprintf("Here are the %d worst things: \n", count))
			buffer.WriteString(formatList(pairs))
			return buffer.String()
		},
	})

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
