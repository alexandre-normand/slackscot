package plugins

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/store"
	"github.com/nlopes/slack"
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
	emojiDelimiter      = ","
)

// Trigger types
const (
	emojiTriggerTypeID    = 'E'
	standardTriggerTypeID = 'S'
)

// triggerType represents a trigger type and holds attributes that
// define it
type triggerType struct {
	ID               rune
	Name             string
	SlackRender      elementRenderer
	ReactionEncoder  reactionEncoder
	ReactionRenderer reactionRenderer
	RegisterRegex    *regexp.Regexp
	DeleteRegex      *regexp.Regexp
}

// elementRenderer is a function that takes in a trigger value and renders it to be included as a line in a table
// for the listTriggers rendering
type elementRenderer func(trigger string, reaction string) (tableRender string)

// renderEmojiTrigger renders an emoji trigger/reaction to be included in a listTriggers output
func renderEmojiTrigger(trigger string, reaction string) (rendered string) {
	return fmt.Sprintf("`%s`\t=> %s", trigger, renderSlackEmojis(parseEmojiList(reaction)))
}

// renderStandardTrigger renders a standard trigger/reaction to be included in a listTriggers output
func renderStandardTrigger(trigger string, reaction string) (rendered string) {
	return fmt.Sprintf("`%s`\t=> `%s`", trigger, reaction)
}

// reactionEncoder is a function that takes in a raw reaction string and encodes it as a string to be persisted
type reactionEncoder func(rawReaction string) (encodedReaction string)

// encodeStandardReaction encodes the rawReaction by just returning the value unchanged (no extra processing necessary)
func encodeStandardReaction(rawReaction string) (encodedReaction string) {
	return rawReaction
}

// encodeEmojiReaction encodes the rawReaction for an emoji trigger by parsing out emoji names and rendering them as a comma-delimited list
func encodeEmojiReaction(rawReaction string) (encodedReaction string) {
	emojis := parseAllEmojis(rawReaction)
	return encodeEmojis(emojis)
}

// reactionRenderer is a function that takes in an encoded reaction and renders it for slack output
type reactionRenderer func(encodedReaction string) (slackRender string)

// encodeStandardReaction encodes the standard encodedReaction by wrapping it with backticks
func renderStandardReaction(encodedReaction string) (slackRender string) {
	return fmt.Sprintf("`%s`", encodedReaction)
}

// renderEmojiReaction encodes the encodedReaction by wrapping each value with colons so that slack renders emojis properly
func renderEmojiReaction(encodedReaction string) (slackRender string) {
	emojis := parseEmojiList(encodedReaction)
	return renderSlackEmojis(emojis)
}

var triggerTypes map[rune]triggerType
var emojiRegex = regexp.MustCompile(":([\\w_-]+):")

func init() {
	registerTriggerRegex := regexp.MustCompile("(?i)\\Atrigger on (.+) with (.+)")
	deleteTriggerRegex := regexp.MustCompile("(?i)\\Aforget trigger on (.+)")

	registerEmojiTriggerRegex := regexp.MustCompile("(?i)\\Aemoji trigger on (.+) with (.+)")
	deleteEmojiTriggerRegex := regexp.MustCompile("(?i)\\Aforget emoji trigger on (.+)")

	triggerTypes = make(map[rune]triggerType)
	triggerTypes[emojiTriggerTypeID] = triggerType{ID: emojiTriggerTypeID, Name: "emoji", SlackRender: renderEmojiTrigger, ReactionEncoder: encodeEmojiReaction, ReactionRenderer: renderEmojiReaction, RegisterRegex: registerEmojiTriggerRegex, DeleteRegex: deleteEmojiTriggerRegex}
	triggerTypes[standardTriggerTypeID] = triggerType{ID: standardTriggerTypeID, Name: "standard", SlackRender: renderStandardTrigger, ReactionEncoder: encodeStandardReaction, ReactionRenderer: renderStandardReaction, RegisterRegex: registerTriggerRegex, DeleteRegex: deleteTriggerRegex}
}

// NewTriggerer creates a new instance of the Triggerer plugin
func NewTriggerer(strStorer store.StringStorer) (triggerer *Triggerer) {
	t := new(Triggerer)

	hearActions := []slackscot.ActionDefinition{
		{
			Hidden:      true,
			Match:       t.matchTriggers,
			Usage:       "say something that includes the trigger",
			Description: "Stays alert to react on registered triggers and react accordingly",
			Answer:      t.reactOnTriggers,
		}}

	commands := pluginCommands(t)

	t.Plugin = slackscot.Plugin{Name: triggererPluginName, Commands: commands, HearActions: hearActions}
	t.triggerStorer = strStorer

	return t
}

// pluginCommands assembles the commands for the plugin
func pluginCommands(t *Triggerer) []slackscot.ActionDefinition {
	return []slackscot.ActionDefinition{
		{
			Hidden:      false,
			Match:       matchNewStandardTrigger,
			Usage:       "trigger on <trigger string> with <reaction string>",
			Description: "Register a trigger which will instruct me to react with `reaction string` when someone says `trigger string`",
			Answer:      t.registerStandardTrigger,
		},
		{
			Hidden:      false,
			Match:       matchDeleteStandardTrigger,
			Usage:       "forget trigger on <trigger string>",
			Description: "Delete a trigger on `trigger string`",
			Answer:      t.deleteStandardTrigger,
		},
		{
			Hidden: true,
			Match: func(m *slackscot.IncomingMessage) bool {
				return strings.HasPrefix(m.NormalizedText, "list triggers")
			},
			Usage:       "list triggers",
			Description: "Lists all registered triggers",
			Answer:      t.listStandardTriggers,
		},
		{
			Hidden:      false,
			Match:       matchNewEmojiTrigger,
			Usage:       "emoji trigger on <trigger string> with <reaction emojis>",
			Description: "Register an emoji trigger which will instruct me to emoji react with `reaction emojis` when someone says `trigger string`",
			Answer:      t.registerEmojiTrigger,
		},
		{
			Hidden:      false,
			Match:       matchDeleteEmojiTrigger,
			Usage:       "forget emoji trigger on <trigger string>",
			Description: "Delete an emoji trigger on `trigger string`",
			Answer:      t.deleteEmojiTrigger,
		},
		{
			Hidden: true,
			Match: func(m *slackscot.IncomingMessage) bool {
				return strings.HasPrefix(m.NormalizedText, "list emoji triggers")
			},
			Usage:       "list emoji triggers",
			Description: "Lists all registered emoji triggers",
			Answer:      t.listEmojiTriggers,
		},
	}
}

// matchNewTrigger returns true if the message matches the trigger registration regex
func matchNewTrigger(m *slackscot.IncomingMessage, triggerTypeID rune) bool {
	return triggerTypes[triggerTypeID].RegisterRegex.MatchString(m.NormalizedText)
}

// matchNewStandardTrigger returns true if the message matches the standard trigger registration regex
func matchNewStandardTrigger(m *slackscot.IncomingMessage) bool {
	return matchNewTrigger(m, standardTriggerTypeID)
}

// matchNewEmojiTrigger returns true if the message matches the emoji trigger registration regex
func matchNewEmojiTrigger(m *slackscot.IncomingMessage) bool {
	return matchNewTrigger(m, emojiTriggerTypeID)
}

// matchDeleteTrigger returns true if the message matches the delete trigger regex
func matchDeleteTrigger(m *slackscot.IncomingMessage, triggerTypeID rune) bool {
	return triggerTypes[triggerTypeID].DeleteRegex.MatchString(m.NormalizedText)
}

// matchDeleteStandardTrigger returns true if the message matches the delete standard trigger regex
func matchDeleteStandardTrigger(m *slackscot.IncomingMessage) bool {
	return matchDeleteTrigger(m, standardTriggerTypeID)
}

// matchDeleteEmojiTrigger returns true if the message matches the delete emoji trigger regex
func matchDeleteEmojiTrigger(m *slackscot.IncomingMessage) bool {
	return matchDeleteTrigger(m, emojiTriggerTypeID)
}

// matchTriggers returns true if the message matches one of the registered triggers
func (t *Triggerer) matchTriggers(m *slackscot.IncomingMessage) bool {
	triggersByType, err := t.getTriggersByType()
	if err != nil {
		t.Logger.Printf("Error loading triggers: %v", err)
		return false
	}

	for _, triggers := range triggersByType {
		for trigger, _ := range triggers {
			if strings.Contains(m.NormalizedText, trigger) {
				return true
			}
		}
	}

	return false
}

// reactOnTrigger reacts on emoji and standard triggers. For standard triggers, only the first match applies. For emoji triggers,
// all matching triggers apply. Note that both emoji triggers and a standard trigger can apply to the same message
func (t *Triggerer) reactOnTriggers(m *slackscot.IncomingMessage) *slackscot.Answer {
	triggersByType, err := t.getTriggersByType()
	if err != nil {
		t.Logger.Printf("Error loading triggers: %v", err)
		return nil
	}

	t.reactOnEmojiTriggers(m, triggersByType[emojiTriggerTypeID])
	return t.reactOnStandardTriggers(m, triggersByType[standardTriggerTypeID])
}

// reactOnStandardTriggers returns a reaction string if it finds a trigger match. Note that only at most one standard trigger can match
func (t *Triggerer) reactOnStandardTriggers(m *slackscot.IncomingMessage, standardTriggers map[string]string) *slackscot.Answer {
	for trigger, reaction := range standardTriggers {
		if strings.Contains(m.NormalizedText, trigger) {
			return &slackscot.Answer{Text: reaction}
		}
	}

	return nil
}

// reactOnEmojiTriggers adds emoji reactions matching emoji triggers, as appropriate
func (t *Triggerer) reactOnEmojiTriggers(m *slackscot.IncomingMessage, emojiTriggers map[string]string) {
	for trigger, reaction := range emojiTriggers {
		if strings.Contains(m.NormalizedText, trigger) {
			for _, emoji := range parseEmojiList(reaction) {
				t.EmojiReactor.AddReaction(emoji, slack.NewRefToMessage(m.Channel, m.Timestamp))
			}
		}
	}
}

// registerTrigger adds or updates a trigger
func (t *Triggerer) registerTrigger(m *slackscot.IncomingMessage, triggerTypeID rune) *slackscot.Answer {
	triggerType := triggerTypes[triggerTypeID]
	trigger, rawReaction := parseRegisterCommand(m.NormalizedText, triggerType.RegisterRegex)
	encodedTrigger := encodeTriggerWithTypeID(trigger, triggerTypeID)
	encodedReaction := triggerType.ReactionEncoder(rawReaction)
	renderedReaction := triggerType.ReactionRenderer(encodedReaction)

	answerMsg := fmt.Sprintf("Registered new %s trigger [`%s` => %s]", triggerType.Name, trigger, renderedReaction)

	encodedExistingReaction, err := t.triggerStorer.GetString(encodedTrigger)
	if encodedExistingReaction != "" {
		existingReactionRender := triggerType.ReactionRenderer(encodedExistingReaction)
		answerMsg = fmt.Sprintf("Replaced %s trigger reaction for [`%s`] with [%s] (was [%s] previously)", triggerType.Name, trigger, renderedReaction, existingReactionRender)
	}

	// Store new/updated trigger
	err = t.triggerStorer.PutString(encodedTrigger, encodedReaction)
	if err != nil {
		answerMsg = fmt.Sprintf("Error persisting %s trigger [`%s` => %s]: `%s`", triggerType.Name, trigger, renderedReaction, err.Error())
		t.Logger.Printf("[%s] %s", triggererPluginName, answerMsg)

		return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
	}

	t.Logger.Debugf("[%s] %s", triggererPluginName, answerMsg)

	return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
}

// registerStandardTrigger adds or updates a standard trigger
func (t *Triggerer) registerStandardTrigger(m *slackscot.IncomingMessage) *slackscot.Answer {
	return t.registerTrigger(m, standardTriggerTypeID)
}

// registerEmojiTrigger adds or updates an emoji trigger
func (t *Triggerer) registerEmojiTrigger(m *slackscot.IncomingMessage) *slackscot.Answer {
	return t.registerTrigger(m, emojiTriggerTypeID)
}

// encodeTriggerWithType encodes a trigger with its type
func encodeTriggerWithTypeID(trigger string, triggerTypeID rune) string {
	var b strings.Builder

	b.WriteRune(triggerTypeID)
	b.WriteString(trigger)

	return b.String()
}

// parseRegisterCommand parses the trigger and reaction from a command string
func parseRegisterCommand(text string, registerRegex *regexp.Regexp) (trigger string, rawReaction string) {
	matches := registerRegex.FindAllStringSubmatch(text, -1)[0]

	trigger = strings.Trim(matches[1], " ")
	reaction := strings.Trim(matches[2], " ")

	return trigger, reaction
}

// renderSlackEmojis renders an array of emojis to a slack string using the :emoji: format
func renderSlackEmojis(emojis []string) (slackRenderedEmojis string) {
	renderedEmojis := make([]string, 0)

	for _, emojiName := range emojis {
		renderedEmojis = append(renderedEmojis, renderSlackEmoji(emojiName))
	}

	return strings.Join(renderedEmojis, ", ")
}

// renderSlackEmoji renders an emoji name (i.e. cat) into a slack emoji string (i.e. :cat:)
func renderSlackEmoji(emojiName string) (slackEmoji string) {
	var b strings.Builder

	b.WriteRune(':')
	b.WriteString(emojiName)
	b.WriteRune(':')

	return b.String()
}

// parseEmojiList returns an array of emojis for a comma-delimited list of emoji names (i.e. cat,dog,wave)
func parseEmojiList(emojiList string) (emojis []string) {
	return strings.Split(emojiList, emojiDelimiter)
}

// encodeEmojis returns a string representation of an array of emojis for storage (i.e. []string{"cat", "dog", "wave"} results in "cat,dog,wave")
func encodeEmojis(emojis []string) (encodedEmojiList string) {
	return strings.Join(emojis, emojiDelimiter)
}

// parseAllEmojis parses a reaction string and finds all emojis
func parseAllEmojis(reaction string) (emojis []string) {
	matches := emojiRegex.FindAllStringSubmatch(reaction, -1)
	emojis = make([]string, 0)

	for _, emojiMatch := range matches {
		emoji := emojiMatch[1]
		emojis = append(emojis, emoji)
	}

	return emojis
}

// deleteStandardTrigger deletes a trigger
func (t *Triggerer) deleteStandardTrigger(m *slackscot.IncomingMessage) *slackscot.Answer {
	return t.deleteTrigger(m, standardTriggerTypeID)
}

func (t *Triggerer) deleteTrigger(m *slackscot.IncomingMessage, triggerTypeID rune) *slackscot.Answer {
	triggerType := triggerTypes[triggerTypeID]
	matches := triggerType.DeleteRegex.FindAllStringSubmatch(m.NormalizedText, -1)[0]
	trigger := strings.Trim(matches[1], " ")

	encodedTrigger := encodeTriggerWithTypeID(trigger, triggerType.ID)
	existingEncodedReaction, err := t.triggerStorer.GetString(encodedTrigger)
	if existingEncodedReaction != "" {
		existingReactionRender := triggerType.ReactionRenderer(existingEncodedReaction)

		// Delete trigger
		err = t.triggerStorer.DeleteString(encodedTrigger)
		if err != nil {
			answerMsg := fmt.Sprintf("Error removing %s trigger [`%s` => %s]: `%s`", triggerType.Name, trigger, existingReactionRender, err.Error())
			t.Logger.Printf("[%s] %s", triggererPluginName, answerMsg)

			return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
		}

		answerMsg := fmt.Sprintf("Deleted %s trigger [`%s` => %s]", triggerType.Name, trigger, existingReactionRender)
		t.Logger.Debugf("[%s] %s", triggererPluginName, answerMsg)

		return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
	} else {
		answerMsg := fmt.Sprintf("No %s trigger found on `%s`", triggerType.Name, trigger)
		return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
	}
}

// deleteEmojiTrigger deletes an emoji trigger
func (t *Triggerer) deleteEmojiTrigger(m *slackscot.IncomingMessage) *slackscot.Answer {
	return t.deleteTrigger(m, emojiTriggerTypeID)
}

// listStandardTriggers returns a message with the full list of registered triggers
func (t *Triggerer) listStandardTriggers(m *slackscot.IncomingMessage) *slackscot.Answer {
	return t.listTriggers("Here are the current triggers: \n", standardTriggerTypeID)
}

// listEmojiTriggers returns a message with the full list of registered triggers
func (t *Triggerer) listEmojiTriggers(m *slackscot.IncomingMessage) *slackscot.Answer {
	return t.listTriggers("Here are the current emoji triggers: \n", emojiTriggerTypeID)
}

// listTriggers renders a list of triggers in a table contained in a code block
func (t *Triggerer) listTriggers(header string, triggerTypeID rune) *slackscot.Answer {
	triggerType := triggerTypes[triggerTypeID]

	triggersByType, err := t.getTriggersByType()
	if err != nil {
		t.Logger.Printf("Error loading triggers: %v", err)
		return &slackscot.Answer{Text: fmt.Sprintf("Error loading triggers:\n```%s```", err.Error()), Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
	}

	triggers := triggersByType[triggerTypeID]

	var buffer bytes.Buffer

	buffer.WriteString(header)
	buffer.WriteString(formatTriggers(triggers, triggerType.SlackRender))
	return &slackscot.Answer{Text: buffer.String(), Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
}

// getTriggers returns all triggers by trigger type. All trigger types are processed
// and callers can safely assume that an entry exists in the returned map for all types even
// if no triggers exists for it (this would be an empty map of triggers => reaction for that type)
func (t *Triggerer) getTriggersByType() (byType map[rune]map[string]string, err error) {
	triggers, err := t.triggerStorer.Scan()
	if err != nil {
		return nil, err
	}

	byType = make(map[rune]map[string]string)
	// Initialize maps for all trigger types
	for triggerTypeID, _ := range triggerTypes {
		byType[triggerTypeID] = make(map[string]string)
	}

	for rawTrigger, reaction := range triggers {
		if len(rawTrigger) > 0 {
			triggerAsRunes := []rune(rawTrigger)
			triggerType := triggerAsRunes[0]
			triggerValue := triggerAsRunes[1:]
			byType[triggerType][string(triggerValue)] = reaction
		}
	}

	return byType, nil
}

// formatTriggers formats the list of triggers
func formatTriggers(triggers map[string]string, render elementRenderer) string {
	keys := make([]string, 0)
	for k := range triggers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b bytes.Buffer
	w := new(tabwriter.Writer)
	bufw := bufio.NewWriter(&b)
	w.Init(bufw, 5, 0, 1, ' ', 0)
	for _, trigger := range keys {
		fmt.Fprintf(w, "\t∙ %s\n", render(trigger, triggers[trigger]))
	}
	fmt.Fprintf(w, "\n")

	bufw.Flush()
	w.Flush()
	return b.String()
}
