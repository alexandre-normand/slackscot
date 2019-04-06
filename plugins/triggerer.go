package plugins

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/actions"
	"github.com/alexandre-normand/slackscot/plugin"
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
	*slackscot.Plugin
	triggerStorer  store.GlobalSiloStringStorer
	triggerRegexes map[string]*regexp.Regexp
}

const (
	// TriggererPluginName holds identifying name for the triggerer plugin
	TriggererPluginName = "triggerer"
	emojiDelimiter      = ","
	globalSiloName      = ""
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
	return fmt.Sprintf("`%s`\t=> %s", trigger, renderStandardReaction(reaction))
}

// reactionEncoder is a function that takes in a raw reaction string and encodes it as a string to be persisted
type reactionEncoder func(rawReaction string) (encodedReaction string, err error)

// encodeStandardReaction encodes the rawReaction by just returning the value unchanged (no extra processing necessary)
func encodeStandardReaction(rawReaction string) (encodedReaction string, err error) {
	return rawReaction, nil
}

// encodeEmojiReaction encodes the rawReaction for an emoji trigger by parsing out emoji names and
// rendering them as a comma-delimited list. If no emojis are found in the rawReaction, an error is returned
func encodeEmojiReaction(rawReaction string) (encodedReaction string, err error) {
	emojis := parseAllEmojis(rawReaction)

	if len(emojis) == 0 {
		return "", fmt.Errorf("`<reaction emojis>` doesn't include any emojis")
	}

	return encodeEmojis(emojis), nil
}

// reactionRenderer is a function that takes in an encoded reaction and renders it for slack output
type reactionRenderer func(encodedReaction string) (slackRender string)

// encodeStandardReaction encodes the standard encodedReaction by wrapping it with backticks except
// when the reaction already includes at least one. In which case, the reaction is returned as-is
func renderStandardReaction(encodedReaction string) (slackRender string) {
	if strings.Contains(encodedReaction, "`") {
		return encodedReaction
	}

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
	registerTriggerRegex := regexp.MustCompile("(?msi)\\Atrigger (anywhere )?on (.+) with (.+)")
	deleteTriggerRegex := regexp.MustCompile("(?i)\\Aforget trigger on (.+)")

	registerEmojiTriggerRegex := regexp.MustCompile("(?i)\\Aemoji trigger (anywhere )?on (.+) with (.+)")
	deleteEmojiTriggerRegex := regexp.MustCompile("(?i)\\Aforget emoji trigger on (.+)")

	triggerTypes = make(map[rune]triggerType)
	triggerTypes[emojiTriggerTypeID] = triggerType{ID: emojiTriggerTypeID, Name: "emoji", SlackRender: renderEmojiTrigger, ReactionEncoder: encodeEmojiReaction, ReactionRenderer: renderEmojiReaction, RegisterRegex: registerEmojiTriggerRegex, DeleteRegex: deleteEmojiTriggerRegex}
	triggerTypes[standardTriggerTypeID] = triggerType{ID: standardTriggerTypeID, Name: "standard", SlackRender: renderStandardTrigger, ReactionEncoder: encodeStandardReaction, ReactionRenderer: renderStandardReaction, RegisterRegex: registerTriggerRegex, DeleteRegex: deleteTriggerRegex}
}

// NewTriggerer creates a new instance of the Triggerer plugin
func NewTriggerer(storer store.GlobalSiloStringStorer) (p *slackscot.Plugin) {
	t := new(Triggerer)
	t.triggerStorer = storer
	t.triggerRegexes = make(map[string]*regexp.Regexp)

	t.Plugin = plugin.New(TriggererPluginName).
		WithHearAction(
			actions.New().Hidden().WithMatcher(t.matchTriggers).WithUsage("say something that includes the trigger").WithDescription("Stays alert to react on registered triggers and react accordingly").WithAnswerer(t.reactOnTriggers).Build(),
		).
		WithCommand(
			actions.New().WithMatcher(matchNewStandardTrigger).WithUsage("trigger [anywhere] on <trigger string> with <reaction string>").WithDescription("Register a trigger which will instruct me to react with `reaction string` when someone says `trigger string`").WithAnswerer(t.registerStandardTrigger).Build(),
		).
		WithCommand(
			actions.New().WithMatcher(matchDeleteStandardTrigger).WithUsage("forget trigger on <trigger string>").WithDescription("Delete a trigger on `trigger string`").WithAnswerer(t.deleteStandardTrigger).Build(),
		).
		WithCommand(actions.New().
			WithMatcher(func(m *slackscot.IncomingMessage) bool {
				return strings.HasPrefix(m.NormalizedText, "list triggers")
			}).
			WithUsage("list triggers").
			WithDescription("Lists all registered triggers").
			WithAnswerer(t.listStandardTriggers).
			Build()).
		WithCommand(
			actions.New().WithMatcher(matchNewEmojiTrigger).WithUsage("emoji trigger [anywhere] on <trigger string> with <reaction emojis>").WithDescription("Register an emoji trigger which will instruct me to emoji react with `reaction emojis` when someone says `trigger string`").WithAnswerer(t.registerEmojiTrigger).Build(),
		).
		WithCommand(
			actions.New().WithMatcher(matchDeleteEmojiTrigger).WithUsage("forget emoji trigger on <trigger string>").WithDescription("Delete an emoji trigger on `trigger string`").WithAnswerer(t.deleteEmojiTrigger).Build(),
		).
		WithCommand(actions.New().
			WithMatcher(func(m *slackscot.IncomingMessage) bool {
				return strings.HasPrefix(m.NormalizedText, "list emoji triggers")
			}).
			WithUsage("list emoji triggers").
			WithDescription("Lists all registered emoji triggers").
			WithAnswerer(t.listEmojiTriggers).
			Build(),
		).
		Build()

	return t.Plugin
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
	triggersByType, err := t.getTriggersByType(m.Channel)
	if err != nil {
		t.Logger.Printf("Error loading triggers: %v", err)
		return false
	}

	for typeID, triggers := range triggersByType {
		for trigger := range triggers {
			exp, err := t.getTriggerRegexp(typeID, trigger)
			if err != nil {
				t.Logger.Printf("Error getting regexp for trigger [%s]: %v", trigger, err)
			}

			if exp.MatchString(m.NormalizedText) {
				return true
			}
		}
	}

	return false
}

// getTriggerRegexp generates a regexp for a given trigger. The resulting regexp is lazily cached so
// the regexes don't have to be recompiled every time
func (t *Triggerer) getTriggerRegexp(triggerTypeID rune, trigger string) (exp *regexp.Regexp, err error) {
	encTrigger := encodeTriggerWithTypeID(trigger, triggerTypeID)
	if exp, ok := t.triggerRegexes[encTrigger]; ok {
		return exp, nil
	}

	t.triggerRegexes[encTrigger], err = regexp.Compile(fmt.Sprintf("(?i)\\b%s\\b", regexp.QuoteMeta(trigger)))
	if err != nil {
		return nil, err
	}

	return t.triggerRegexes[encTrigger], nil
}

// reactOnTrigger reacts on emoji and standard triggers. For standard triggers, only the first match applies. For emoji triggers,
// all matching triggers apply. Note that both emoji triggers and a standard trigger can apply to the same message
func (t *Triggerer) reactOnTriggers(m *slackscot.IncomingMessage) *slackscot.Answer {
	triggersByType, err := t.getTriggersByType(m.Channel)
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
		exp, err := t.getTriggerRegexp(standardTriggerTypeID, trigger)
		if err != nil {
			t.Logger.Printf("Error getting regexp for trigger [%s]: %v", trigger, err)
		}

		if exp.MatchString(m.NormalizedText) {
			return &slackscot.Answer{Text: reaction}
		}
	}

	return nil
}

// reactOnEmojiTriggers adds emoji reactions matching emoji triggers, as appropriate
func (t *Triggerer) reactOnEmojiTriggers(m *slackscot.IncomingMessage, emojiTriggers map[string]string) {
	for trigger, reaction := range emojiTriggers {
		exp, err := t.getTriggerRegexp(emojiTriggerTypeID, trigger)
		if err != nil {
			t.Logger.Printf("Error getting regexp for trigger [%s]: %v", trigger, err)
		}

		if exp.MatchString(m.NormalizedText) {
			for _, emoji := range parseEmojiList(reaction) {
				t.EmojiReactor.AddReaction(emoji, slack.NewRefToMessage(m.Channel, m.Timestamp))
			}
		}
	}
}

// registerTrigger adds or updates a trigger
func (t *Triggerer) registerTrigger(m *slackscot.IncomingMessage, triggerTypeID rune) *slackscot.Answer {
	triggerType := triggerTypes[triggerTypeID]
	silo, trigger, rawReaction := parseRegisterCommand(m, triggerType.RegisterRegex)
	encodedTrigger := encodeTriggerWithTypeID(trigger, triggerTypeID)
	encodedReaction, err := triggerType.ReactionEncoder(rawReaction)
	if err != nil {
		return &slackscot.Answer{Text: fmt.Sprintf("Invalid reaction for %s trigger: %s", triggerType.Name, err.Error()), Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
	}

	renderedReaction := triggerType.ReactionRenderer(encodedReaction)
	answerMsg := fmt.Sprintf("Registered new %s trigger [`%s` => %s]", triggerType.Name, trigger, renderedReaction)

	encodedExistingReaction, err := t.triggerStorer.GetSiloString(silo, encodedTrigger)
	if encodedExistingReaction != "" {
		existingReactionRender := triggerType.ReactionRenderer(encodedExistingReaction)
		answerMsg = fmt.Sprintf("Replaced %s trigger reaction for [`%s`] with [%s] (was [%s] previously)", triggerType.Name, trigger, renderedReaction, existingReactionRender)
	}

	// Store new/updated trigger
	err = t.triggerStorer.PutSiloString(silo, encodedTrigger, encodedReaction)
	if err != nil {
		answerMsg = fmt.Sprintf("Error persisting %s trigger [`%s` => %s]: `%s`", triggerType.Name, trigger, renderedReaction, err.Error())
		t.Logger.Printf("[%s] %s", TriggererPluginName, answerMsg)

		return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
	}

	t.Logger.Debugf("[%s] %s", TriggererPluginName, answerMsg)

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

// parseRegisterCommand parses the global mode, trigger and reaction from a command string
func parseRegisterCommand(m *slackscot.IncomingMessage, registerRegex *regexp.Regexp) (silo string, trigger string, rawReaction string) {
	matches := registerRegex.FindAllStringSubmatch(m.NormalizedText, -1)[0]

	where := strings.Trim(matches[1], " ")
	trigger = strings.Trim(matches[2], " ")
	reaction := strings.Trim(matches[3], " ")

	silo = m.Channel
	// if the optional "anywhere" was included in the instruction, set the silo
	// to the global one
	if strings.HasPrefix(where, "anywhere") {
		silo = globalSiloName
	}

	return silo, trigger, reaction
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

// deleteEmojiTrigger deletes an emoji trigger
func (t *Triggerer) deleteEmojiTrigger(m *slackscot.IncomingMessage) *slackscot.Answer {
	return t.deleteTrigger(m, emojiTriggerTypeID)
}

func (t *Triggerer) deleteTrigger(m *slackscot.IncomingMessage, triggerTypeID rune) *slackscot.Answer {
	triggerType := triggerTypes[triggerTypeID]
	matches := triggerType.DeleteRegex.FindAllStringSubmatch(m.NormalizedText, -1)[0]
	trigger := strings.Trim(matches[1], " ")

	a := t.deleteChannelTrigger(m.Channel, trigger, triggerType)
	if a == nil {
		// If there isn't a channel trigger, we assume the intent was to delete a global one so we try that
		a = t.deleteChannelTrigger(globalSiloName, trigger, triggerType)
	}

	if a != nil {
		return a
	}

	answerMsg := fmt.Sprintf("No %s trigger found on `%s`", triggerType.Name, trigger)
	return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
}

// deleteChannelTrigger deletes a trigger for the given channel (which is the silo the trigger is stored in).
// This is meant to allow deleting a trigger for a specific channel but also a global one using the globalSiloName
func (t *Triggerer) deleteChannelTrigger(channel string, trigger string, ttype triggerType) *slackscot.Answer {
	encodedTrigger := encodeTriggerWithTypeID(trigger, ttype.ID)
	existingEncodedReaction, err := t.triggerStorer.GetSiloString(channel, encodedTrigger)
	if existingEncodedReaction != "" {
		existingReactionRender := ttype.ReactionRenderer(existingEncodedReaction)

		// Delete trigger
		err = t.triggerStorer.DeleteSiloString(channel, encodedTrigger)
		if err != nil {
			answerMsg := fmt.Sprintf("Error removing %s trigger [`%s` => %s]: `%s`", ttype.Name, trigger, existingReactionRender, err.Error())
			t.Logger.Printf("[%s] %s", TriggererPluginName, answerMsg)

			return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
		}

		answerMsg := fmt.Sprintf("Deleted %s trigger [`%s` => %s]", ttype.Name, trigger, existingReactionRender)
		t.Logger.Debugf("[%s] %s", TriggererPluginName, answerMsg)

		return &slackscot.Answer{Text: answerMsg, Options: []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}}
	}

	return nil
}

// listStandardTriggers returns a message with the full list of registered triggers
func (t *Triggerer) listStandardTriggers(m *slackscot.IncomingMessage) *slackscot.Answer {
	return t.listTriggers(m.Channel, "Here are the current triggers: \n", standardTriggerTypeID)
}

// listEmojiTriggers returns a message with the full list of registered triggers
func (t *Triggerer) listEmojiTriggers(m *slackscot.IncomingMessage) *slackscot.Answer {
	return t.listTriggers(m.Channel, "Here are the current emoji triggers: \n", emojiTriggerTypeID)
}

// listTriggers renders a list of triggers in a table contained in a code block
func (t *Triggerer) listTriggers(channelID string, header string, triggerTypeID rune) *slackscot.Answer {
	triggerType := triggerTypes[triggerTypeID]

	triggersByType, err := t.getTriggersByType(channelID)
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

// getTriggers returns all triggers by trigger type for a given channel ID. All trigger types are processed
// and callers can safely assume that an entry exists in the returned map for all types even
// if no triggers exists for it (this would be an empty map of triggers => reaction for that type)
func (t *Triggerer) getTriggersByType(channelID string) (byType map[rune]map[string]string, err error) {
	// Start adding global triggers
	triggers, err := t.triggerStorer.ScanSilo(globalSiloName)
	if err != nil {
		return nil, err
	}

	chTriggers, err := t.triggerStorer.ScanSilo(channelID)
	if err != nil {
		return nil, err
	}

	// Add channel-specific triggers, overriding any duplicates so that the channel version
	// wins
	for k, v := range chTriggers {
		triggers[k] = v
	}

	byType = make(map[rune]map[string]string)
	// Initialize maps for all trigger types
	for triggerTypeID := range triggerTypes {
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
		reaction := triggers[trigger]
		fmt.Fprintf(w, "\t• %s\n", render(trigger, reaction))
	}
	fmt.Fprintf(w, "\n")

	bufw.Flush()
	w.Flush()
	return b.String()
}
