// Package assertplugin provides testing functions to validate a plugin's overall functionality.
// This package is designed to play well but not require the assertanswer package for validation
// of answers
//
// Note that all commands and hearActions are evaluated by assertplugin's driver but this is a
// simplified version of how slackscot actually drives plugins and aims to provide the minimal
// processing required to allow a plugin to test functionality given an incoming message.
// Users should take special care to use include <@botUserID> with the same botUserID with which the
// plugin driver has been instantiated in the message text inputs to test commands (or include a
// channel name that starts with D for direct channel testing)
package assertplugin

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/schedule"
	"github.com/alexandre-normand/slackscot/test/capture"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"testing"
)

// Asserter represents a plugin driver/asserter and holds the bot identifier that tests are using when
// sending test messages for processing
type Asserter struct {
	botUserID string
	t         *testing.T
	logger    *log.Logger
}

// New creates a new asserter with the given botUserId
// (only include the id without the '@' prefix).
// The botUserId is used in order to detect commands formed with
// <@botUserId>
func New(t *testing.T, botUserID string, options ...Option) (a *Asserter) {
	a = new(Asserter)
	a.botUserID = botUserID
	a.t = t

	for _, option := range options {
		option(a)
	}

	return a
}

// Option defines an option for the Asserter
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

// ResultWithUploadsValidator is a function to do further validation of the answers, emoji reactions and file uploads
// resulting from a plugin processing of all of its commands and hear actions. The return value is meant to be true
// if validation is successful and false otherwise (following the testify convention)
type ResultWithUploadsValidator func(t *testing.T, answers []*slackscot.Answer, emojis []string, fileUploads []slack.FileUploadParameters) bool

// ScheduleResultValidator is a function to do further validation of the messages sent by a slackscot.ScheduledAction.
// The messages sent during the execution of scheduled actions is given as a map of channel IDs to messages
// sent on that channel. The return value is meant to be true if validation is successful and false otherwise
// (following the testify convention)
type ScheduleResultValidator func(t *testing.T, sentMessagesByChannelID map[string][]string) bool

// AnswersAndReacts drives a plugin and collects Answers as well as emoji reactions. Once all of those have been collected,
// it passes handling to a validator to assert the expected answers and emoji reactions. It follows the style of
// github.com/stretchr/testify/assert as far as returning true/false to indicate success for further nested testing.
func (a *Asserter) AnswersAndReacts(p *slackscot.Plugin, m *slack.Msg, validate ResultValidator) (valid bool) {
	answers, emojis, _ := a.injectServicesAndRun(p, m)

	return validate(a.t, answers, emojis)
}

// AnswersAndReactsWithUploads drives a plugin and collects Answers as well as emoji reactions and file uploads.
// Once all of those have been collected, it passes handling to a validator to assert the expected answers,
// emoji reactions and file uploads. It follows the style of github.com/stretchr/testify/assert as far as
// returning true/false to indicate success for further nested testing.
func (a *Asserter) AnswersAndReactsWithUploads(p *slackscot.Plugin, m *slack.Msg, validate ResultWithUploadsValidator) (valid bool) {
	answers, emojis, fileUploads := a.injectServicesAndRun(p, m)

	return validate(a.t, answers, emojis, fileUploads)
}

// RunsOnSchedule drives a plugin's scheduled actions that match the schedule definition being passed in (i.e. "Every 1 hour" will
// run all actions scheduled to run every hour) and collects all the sent messages. Once all have been collected,
// the results are passed to the ScheduleResultValidator as a map[string][]string where the key is the channel id
// and the value holds the messages sent to that channel
func (a *Asserter) RunsOnSchedule(p *slackscot.Plugin, schedule schedule.Definition, validate ScheduleResultValidator) (valid bool) {
	a.injectServices(p)
	sender := capture.NewRealTimeSender()

	didOneRun := false
	for _, action := range p.ScheduledActions {
		if action.Schedule == schedule {
			action.Action(sender)
			didOneRun = true
		}
	}

	return assert.Truef(a.t, didOneRun, "Expected at least one action to run on schedule [%s] but none did", schedule) && validate(a.t, sender.SentMessages)
}

// DoesNotRunOnSchedule drives a plugin's scheduled actions and validate that none of the
// ScheduledActions run on the specified schedule
func (a *Asserter) DoesNotRunOnSchedule(p *slackscot.Plugin, schedule schedule.Definition) (valid bool) {
	a.injectServices(p)
	sender := capture.NewRealTimeSender()

	for _, action := range p.ScheduledActions {
		if action.Schedule == schedule {
			action.Action(sender)
			return assert.Falsef(a.t, true, "Expected no action to run for schedule [%s] but [%s] did run", schedule, action.Description)
		}
	}

	// No action ran so we can assert that it was indeed false
	return assert.False(a.t, false)
}

// injectServicesAndRun injects services in the plugin, drives all of its actions and returns the answers and captured data
// from the execution
func (a *Asserter) injectServicesAndRun(p *slackscot.Plugin, m *slack.Msg) (answers []*slackscot.Answer, emojis []string, fileUploads []slack.FileUploadParameters) {
	emojiCaptor, fileUploadCaptor := a.injectServices(p)

	answers = a.driveActions(p, m)

	return answers, emojiCaptor.Emojis, fileUploadCaptor.FileUploads
}

func (a *Asserter) injectServices(p *slackscot.Plugin) (emojiCaptor *capture.EmojiReactionCaptor, fileUploadCaptor *capture.FileUploadCaptor) {
	emojiCaptor = capture.NewEmojiReactor()
	p.EmojiReactor = emojiCaptor
	fileUploadCaptor = capture.NewFileUploader()
	p.FileUploader = slackscot.NewFileUploader(fileUploadCaptor)
	p.Logger = slackscot.NewSLogger(getLogger(a), true)

	return emojiCaptor, fileUploadCaptor
}

func getLogger(a *Asserter) (logger *log.Logger) {
	if a.logger != nil {
		return a.logger
	}

	var b strings.Builder
	return log.New(&b, "", 0)
}

func (a *Asserter) driveActions(p *slackscot.Plugin, m *slack.Msg) (answers []*slackscot.Answer) {
	botMentionPrefix := fmt.Sprintf("<@%s> ", a.botUserID)

	if strings.HasPrefix(m.Text, botMentionPrefix) {
		normalizedText := strings.TrimPrefix(m.Text, botMentionPrefix)
		inMsg := slackscot.IncomingMessage{NormalizedText: normalizedText, Msg: *m}

		return runActions(p.Commands, &inMsg)
	}

	inMsg := slackscot.IncomingMessage{NormalizedText: m.Text, Msg: *m}

	if strings.HasPrefix(m.Channel, "D") {
		return runActions(p.Commands, &inMsg)
	}

	return runActions(p.HearActions, &inMsg)
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
