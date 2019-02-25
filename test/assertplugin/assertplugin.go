package assertplugin

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/test"
	"github.com/nlopes/slack"
	"log"
	"strings"
	"testing"
)

// Asserter represents a plugin driver/asserter and holds the bot identifier that tests are using when
// sending test messages for processing
type Asserter struct {
	botUserID string
	logger    *log.Logger
}

// New creates a new asserter with the given botUserId
// (only include the id without the '@' prefix).
// The botUserId is used in order to detect commands formed with
// <@botUserId>
func New(botUserID string, options ...Option) (a *Asserter) {
	a = new(Asserter)
	a.botUserID = botUserID

	for _, option := range options {
		option(a)
	}

	return a
}

type Option func(*Asserter)

// OptionLog sets a logger for the asserter such that this logger is attached to the plugin when driven by
// the asserter
func OptionLog(logger *log.Logger) func(*Asserter) {
	return func(a *Asserter) {
		a.logger = logger
	}
}

// ResultValidator is a function to do further validation of the answers and emoji reactions resulting from
// a plugin processing of all of its commands and hear actions. The return value is meant to be true if validation
// is successful and false otherwise (following the testify convention)
type ResultValidator func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool

// AnswersAndReacts drives a plugin and collects Answers as well as emoji reactions. Once all of those have been collected,
// it passes handling to a validator to assert the expected answers and emoji reactions. It follows the style of
// github.com/stretchr/testify/assert as far as returning true/false to indicate success for further nested testing.
//
// Note that all commands and hearActions are evaluated but this is a simplified version of how slackscot actually drives
// plugins and aims to provide the minimal processing required to allow a plugin to test functionality given an
// incoming message. Users should take special care to use include <@botUserID> with the same botUserID with which the
// plugin driver has been instantiated in the message text inputs to test commands (or include a channel name that
// starts with D for direct channel testing)
func (a *Asserter) AnswersAndReacts(t *testing.T, p *slackscot.Plugin, m *slack.Msg, validate ResultValidator) (valid bool) {
	ec := test.NewEmojiReactionCaptor()
	p.EmojiReactor = ec

	// Attach asserter logger or use a default one that logs to a string builder
	if a.logger != nil {
		p.Logger = slackscot.NewSLogger(a.logger, true)
	} else {
		var b strings.Builder
		p.Logger = slackscot.NewSLogger(log.New(&b, "", 0), true)
	}

	answers := a.driveActions(p, m)

	return validate(t, answers, ec.Emojis)
}

func (a *Asserter) driveActions(p *slackscot.Plugin, m *slack.Msg) (answers []*slackscot.Answer) {
	botMentionPrefix := fmt.Sprintf("<@%s> ", a.botUserID)

	if strings.HasPrefix(m.Text, botMentionPrefix) {
		normalizedText := strings.TrimPrefix(m.Text, botMentionPrefix)
		inMsg := slackscot.IncomingMessage{NormalizedText: normalizedText, Msg: *m}

		return runActions(p.Commands, &inMsg)
	} else {
		inMsg := slackscot.IncomingMessage{NormalizedText: m.Text, Msg: *m}

		if strings.HasPrefix(m.Channel, "D") {
			return runActions(p.Commands, &inMsg)
		} else {
			return runActions(p.HearActions, &inMsg)
		}
	}
}

func runActions(actions []slackscot.ActionDefinition, m *slackscot.IncomingMessage) (answers []*slackscot.Answer) {
	answers = make([]*slackscot.Answer, 0)

	for _, action := range actions {
		if action.Match(m) {
			a := action.Answer(m)

			if a != nil {
				answers = append(answers, a)
			}
		}
	}

	return answers
}
