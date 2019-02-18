package plugins

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/store"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
)

// Triggerer holds the plugin data for the triggerer plugin
// The triggerer plugin consists of a command to register a new trigger with its answer along
// with a hear action that will listen and react if one of the registered triggers is said
type Triggerer struct {
	slackscot.Plugin
	triggerStorer store.StringStorer
}

const (
	// triggererPluginName holds identifying name for the triggerer plugin
	triggererPluginName = "triggerer"
)

var registerTriggerRegex = regexp.MustCompile("(?i)\\Atrigger on (.+) with (.+)")
var deleteTriggerRegex = regexp.MustCompile("(?i)\\Aforget trigger on (.+)")

// NewTriggerer creates a new instance of the Triggerer plugin
func NewTriggerer(strStorer store.StringStorer) (triggerer *Triggerer) {
	t := new(Triggerer)

	hearActions := []slackscot.ActionDefinition{
		{
			Hidden:      true,
			Match:       t.matchTriggers,
			Usage:       "say something that includes the trigger",
			Description: "Stays alert to react on registered triggers and react accordingly",
			Answer:      t.reactOnTrigger,
		}}

	commands := []slackscot.ActionDefinition{
		{
			Hidden:      false,
			Match:       matchNewTrigger,
			Usage:       "trigger on <trigger string> with <reaction string>",
			Description: "Register a trigger which will instruct me to react with `reaction string` when someone says `trigger string`",
			Answer:      t.registerTrigger,
		},
		{
			Hidden:      false,
			Match:       matchDeleteTrigger,
			Usage:       "forget trigger on <trigger string>",
			Description: "Delete a trigger on `trigger string`",
			Answer:      t.deleteTrigger,
		},
		{
			Hidden: true,
			Match: func(m *slackscot.IncomingMessage) bool {
				return strings.HasPrefix(m.NormalizedText, "list triggers")
			},
			Usage:       "list triggers",
			Description: "Lists all registered triggers",
			Answer:      t.listTriggers,
		},
	}

	t.Plugin = slackscot.Plugin{Name: triggererPluginName, Commands: commands, HearActions: hearActions}
	t.triggerStorer = strStorer

	return t
}

// matchNewTrigger returns true if the message matches the trigger registration regex
func matchNewTrigger(m *slackscot.IncomingMessage) bool {
	return registerTriggerRegex.MatchString(m.NormalizedText)
}

// matchDeleteTrigger returns true if the message matches the delete trigger regex
func matchDeleteTrigger(m *slackscot.IncomingMessage) bool {
	return deleteTriggerRegex.MatchString(m.NormalizedText)
}

// matchTriggers returns true if the message matches one of the registerer triggers
func (t *Triggerer) matchTriggers(m *slackscot.IncomingMessage) bool {
	triggers, err := t.triggerStorer.Scan()
	if err != nil {
		t.Logger.Printf("Error loading triggers: %v", err)
		return false
	}

	for trigger, _ := range triggers {
		if strings.Contains(m.NormalizedText, trigger) {
			return true
		}
	}

	return false
}

// matchTriggers returns true if the message matches one of the registerer triggers
func (t *Triggerer) reactOnTrigger(m *slackscot.IncomingMessage) *slackscot.Answer {
	triggers, err := t.triggerStorer.Scan()
	if err != nil {
		t.Logger.Printf("Error loading triggers: %v", err)
		return &slackscot.Answer{Text: ""}
	}

	for trigger, reaction := range triggers {
		if strings.Contains(m.NormalizedText, trigger) {
			return &slackscot.Answer{Text: reaction}
		}
	}

	return &slackscot.Answer{Text: ""}
}

// registerTrigger adds or updates a trigger
func (t *Triggerer) registerTrigger(m *slackscot.IncomingMessage) *slackscot.Answer {
	trigger, reaction := parseRegisterCommand(m.NormalizedText)

	answerMsg := fmt.Sprintf("Registered new trigger `[%s => %s]`", trigger, reaction)

	existingReaction, err := t.triggerStorer.GetString(trigger)
	if existingReaction != "" {
		answerMsg = fmt.Sprintf("Replaced trigger reaction for [`%s`] with [`%s`] (was [`%s`] previously)", trigger, reaction, existingReaction)
	}

	// Store new/updated trigger
	err = t.triggerStorer.PutString(trigger, reaction)
	if err != nil {
		answerMsg = fmt.Sprintf("Error persisting trigger `[%s => %s]`: `%s`", trigger, reaction, err.Error())
		t.Logger.Printf("[%s] %s", triggererPluginName, answerMsg)

		return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
	}

	t.Logger.Debugf("[%s] %s", triggererPluginName, answerMsg)

	return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
}

// parseRegisterCommand parses the trigger and reaction from a command string
func parseRegisterCommand(text string) (trigger string, reaction string) {
	matches := registerTriggerRegex.FindAllStringSubmatch(text, -1)[0]

	if len(matches[1]) > 0 {
		trigger = strings.Trim(matches[1], " ")
		reaction = strings.Trim(matches[2], " ")

		return trigger, reaction
	} else {
		trigger := strings.Trim(matches[3], " ")
		reaction := strings.Trim(matches[4], " ")

		return trigger, reaction
	}
}

// deleteTrigger deletes a trigger
func (t *Triggerer) deleteTrigger(m *slackscot.IncomingMessage) *slackscot.Answer {
	matches := deleteTriggerRegex.FindAllStringSubmatch(m.NormalizedText, -1)[0]
	trigger := strings.Trim(matches[1], " ")

	existingReaction, err := t.triggerStorer.GetString(trigger)
	if existingReaction != "" {
		// Delete trigger
		err = t.triggerStorer.DeleteString(trigger)
		if err != nil {
			answerMsg := fmt.Sprintf("Error removing trigger `[%s => %s]`: `%s`", trigger, existingReaction, err.Error())
			t.Logger.Printf("[%s] %s", triggererPluginName, answerMsg)

			return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
		}

		answerMsg := fmt.Sprintf("Deleted trigger [`%s => %s`]", trigger, existingReaction)
		t.Logger.Debugf("[%s] %s", triggererPluginName, answerMsg)

		return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
	} else {
		answerMsg := fmt.Sprintf("No trigger found on `%s`", trigger)
		return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
	}
}

// listTriggers returns a message with the full list of registered triggers
func (t *Triggerer) listTriggers(message *slackscot.IncomingMessage) *slackscot.Answer {
	triggers, err := t.triggerStorer.Scan()
	if err != nil {
		t.Logger.Printf("Error loading triggers: %v", err)
		return &slackscot.Answer{Text: fmt.Sprintf("Error loading triggers:\n```%s```", err.Error()), Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
	}

	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("Here are the current triggers: \n"))
	buffer.WriteString(t.formatTriggers(triggers))
	return &slackscot.Answer{Text: buffer.String(), Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
}

// formatTriggers formats the list of triggers in a nice table using tabwriter presented in a code block
func (t *Triggerer) formatTriggers(triggers map[string]string) string {
	keys := make([]string, 0)
	for k := range triggers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b bytes.Buffer
	b.WriteString("```")
	w := new(tabwriter.Writer)
	bufw := bufio.NewWriter(&b)
	w.Init(bufw, 5, 0, 1, ' ', 0)
	for _, trigger := range keys {
		fmt.Fprintf(w, "%s\t=> %s\n", trigger, triggers[trigger])
	}
	fmt.Fprintf(w, "```\n")

	bufw.Flush()
	w.Flush()
	return b.String()
}
