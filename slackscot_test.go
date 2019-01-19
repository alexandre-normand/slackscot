package slackscot

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/v2/config"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
)

const (
	botUserID                   = "BotUserID"
	timestamp1                  = "1546833210.036900"
	timestamp2                  = "1546833214.036900"
	firstReplyTimestamp         = 1547785956
	replyTimeIncrementInSeconds = 10
)

type sentMessage struct {
	channelID  string
	msgOptions []slack.MsgOption
}

type updatedMessage struct {
	channelID  string
	timestamp  string
	msgOptions []slack.MsgOption
}

type deletedMessage struct {
	channelID string
	timestamp string
}

type nullWriter struct {
}

func (nw *nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

type inMemoryChatDriver struct {
	timeCursor  uint64
	sentMsgs    []sentMessage
	updatedMsgs []updatedMessage
	deletedMsgs []deletedMessage
}

func (c *inMemoryChatDriver) SendMessage(channelID string, options ...slack.MsgOption) (rChannelID string, rTimestamp string, rText string, err error) {
	c.sentMsgs = append(c.sentMsgs, sentMessage{channelID: channelID, msgOptions: options})
	return channelID, c.nextTimestamp(), fmt.Sprintf("Message on %s", channelID), nil
}

func (c *inMemoryChatDriver) UpdateMessage(channelID, timestamp string, options ...slack.MsgOption) (rChannelID string, rTimestamp string, rText string, err error) {
	c.updatedMsgs = append(c.updatedMsgs, updatedMessage{channelID: channelID, timestamp: timestamp, msgOptions: options})
	return channelID, c.nextTimestamp(), fmt.Sprintf("Message updated on %s", channelID), nil
}

func (c *inMemoryChatDriver) DeleteMessage(channelID string, timestamp string) (rChannelID string, rTimestamp string, err error) {
	c.deletedMsgs = append(c.deletedMsgs, deletedMessage{channelID: channelID, timestamp: timestamp})
	return channelID, c.nextTimestamp(), nil
}

func (c *inMemoryChatDriver) nextTimestamp() (fmtTime string) {
	c.timeCursor = c.timeCursor + replyTimeIncrementInSeconds
	return formatTimestamp(c.timeCursor)
}

func formatTimestamp(ts uint64) string {
	return fmt.Sprintf("%d.000000", ts)
}

type infoFinder struct {
}

type userInfoFinder struct {
}

type testPlugin struct {
	Plugin
}

// Option type for building a message with additional options for specific test cases
type testMsgOption func(e *slack.MessageEvent)

func optionChangedMessage(text string, user string, originalTs string) func(e *slack.MessageEvent) {
	return func(e *slack.MessageEvent) {
		e.SubType = "message_changed"
		e.SubMessage = &slack.Msg{Text: text, User: user, Timestamp: originalTs}
	}
}

func optionDeletedMessage(channelID string, timestamp string) func(e *slack.MessageEvent) {
	return func(e *slack.MessageEvent) {
		e.SubType = "message_deleted"
		e.DeletedTimestamp = timestamp
		e.Channel = channelID
	}
}

func optionDirectMessage(botUserID string) func(e *slack.MessageEvent) {
	return func(e *slack.MessageEvent) {
		e.Channel = fmt.Sprintf("D%s", botUserID)
	}
}

func optionPublicMessageToBot(botUserID string, channelID string) func(e *slack.MessageEvent) {
	return func(e *slack.MessageEvent) {
		e.Channel = channelID
		e.Text = fmt.Sprintf("<@%s> %s", botUserID, e.Text)
	}
}

func newTestPlugin() (tp *testPlugin) {
	tp = new(testPlugin)
	tp.Name = "noRules"
	tp.Commands = []ActionDefinition{{
		Match: func(t string, m *slack.Msg) bool {
			return strings.HasPrefix(t, "make")
		},
		Usage:       "make `<something>`",
		Description: "Have the test bot make something for you",
		Answer: func(m *slack.Msg) string {
			return fmt.Sprintf("Make it yourself, @%s", m.User)
		},
	}}
	tp.HearActions = []ActionDefinition{{
		Hidden: true,
		Match: func(t string, m *slack.Msg) bool {
			return strings.Contains(t, "blue jays")
		},
		Usage:       "Talk about my secret topic",
		Description: "Reply with usage instructions",
		Answer: func(m *slack.Msg) string {
			return "I heard you say something about blue jays?"
		},
	}}
	tp.ScheduledActions = nil

	return tp
}

func (i *infoFinder) GetInfo() (user *slack.Info) {
	return &slack.Info{User: &slack.UserDetails{ID: "BotUserID", Name: "Daniel Quinn"}}
}

func (u *userInfoFinder) GetUserInfo(userID string) (user *slack.User, err error) {
	return &slack.User{ID: botUserID, Name: "Daniel Quinn"}, nil
}

func TestLogfileOverrideUsed(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "test")
	assert.Nil(t, err)

	defer os.Remove(tmpfile.Name()) // clean up

	runSlackscotWithIncomingEvents(t, []slack.RTMEvent{}, OptionLogfile(tmpfile))

	logs, err := ioutil.ReadFile(tmpfile.Name())
	assert.Nil(t, err)

	assert.Contains(t, string(logs), "Connection counter: 0")
}

func TestLatencyReport(t *testing.T) {
	_, _, _, logs := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		slack.RTMEvent{Type: "latency_report", Data: &slack.LatencyReport{Value: 120}},
	})

	assert.Contains(t, logs, "Current latency: 120ns")
}

func TestRTMError(t *testing.T) {
	_, _, _, logs := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		slack.RTMEvent{Type: "rtm_error", Data: &slack.RTMError{Code: 500, Msg: "test error"}},
	})

	assert.Contains(t, logs, "Error: Code 500 - test error")
}

func TestInvalidCredentialsShutsdownImmediately(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, logs := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		slack.RTMEvent{Type: "invalid_auth_event", Data: &slack.InvalidAuthEvent{}},
		newRTMMessageEvent(newMessageEvent("Cgeneral", "Bonjour", "Alphonse", timestamp1)),
	})

	assert.Contains(t, logs, "Invalid credentials")
	assert.Equal(t, 0, len(sentMsgs))
	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
}

func TestHandleIncomingMessageTriggeringResponse(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
	})

	assert.Equal(t, 1, len(sentMsgs))
	assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
}

func TestIgnoreReplyToMessage(t *testing.T) {
	msge := new(slack.MessageEvent)
	msge.Type = "message"
	msge.Channel = "CHGENERAL"
	msge.User = "Alphone"
	msge.Text = "blue jars"
	msge.ReplyTo = 1

	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		newRTMMessageEvent(msge),
	})

	assert.Equal(t, 0, len(sentMsgs))
	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
}

func TestIncomingMessageUpdateTriggeringResponseUpdate(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Ignored", timestamp2, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
	}

	if assert.Equal(t, 1, len(updatedMsgs)) {
		assert.Equal(t, 3, len(updatedMsgs[0].msgOptions))
	}

	assert.Equal(t, 0, len(deletedMsgs))
}

func TestIncomingMessageTriggeringNewResponse(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "nothing important", "Alphonse", timestamp1)),
		// This message update should now trigger the hear action
		newRTMMessageEvent(newMessageEvent("Cgeneral", "nothing important", "Ignored", timestamp2, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
}

func TestIncomingTriggeringMessageUpdatedToNotTriggerAnymore(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp2, optionChangedMessage("never mind", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	if assert.Equal(t, 1, len(deletedMsgs)) {
		assert.Equal(t, deletedMessage{channelID: "Cgeneral", timestamp: formatTimestamp(firstReplyTimestamp)}, deletedMsgs[0])
	}
}

func TestDirectMessageMatchingCommand(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		// Trigger the command action
		newRTMMessageEvent(newMessageEvent("DFromUser", "make me happy", "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
}

func TestDirectMessageNotMatchingAnything(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		// Trigger the command action
		newRTMMessageEvent(newMessageEvent("DFromUser", "hey you", "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
}

func TestAtMessageNotMatchingAnything(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		// At Message but not matching the command
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("<@%s> hey you", botUserID), "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
}

func TestIncomingTriggeringMessageUpdatedToTriggerDifferentAction(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		// Trigger the hear action
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		// Update the message to now trigger the command instead of the hear action
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp2, optionChangedMessage(fmt.Sprintf("<@%s> make me laugh", botUserID), "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 2, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
	}

	assert.Equal(t, 0, len(updatedMsgs))

	if assert.Equal(t, 1, len(deletedMsgs)) {
		assert.Equal(t, deletedMessage{channelID: "Cgeneral", timestamp: formatTimestamp(firstReplyTimestamp)}, deletedMsgs[0])
	}
}

func TestTriggeringMessageDeletion(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Ignored", timestamp2, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
	}

	if assert.Equal(t, 1, len(updatedMsgs)) {
		assert.Equal(t, 3, len(updatedMsgs[0].msgOptions))
	}

	assert.Equal(t, 0, len(deletedMsgs))
}

func TestIncomingMessageUpdateTriggeringResponseDeletion(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp2, optionDeletedMessage("Cgeneral", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	if assert.Equal(t, 1, len(deletedMsgs)) {
		assert.Equal(t, deletedMessage{channelID: "Cgeneral", timestamp: formatTimestamp(firstReplyTimestamp)}, deletedMsgs[0])
	}
}

func TestIncomingMessageNotTriggeringResponse(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "bonjour", "Alphonse", timestamp1)),
	})

	assert.Equal(t, 0, len(sentMsgs))
	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
}

func TestIncomingMessageFromOurselfIgnored(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays are cool", botUserID, timestamp1)),
	})

	assert.Equal(t, 0, len(sentMsgs))
	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
}

func TestNewWithInvalidResponseCacheSize(t *testing.T) {
	v := config.NewViperWithDefaults()
	v.Set(config.ResponseCacheSizeKey, -1)

	_, err := NewSlackscot("chicadee", v)
	assert.NotNil(t, err)
}

func newRTMMessageEvent(msgEvent *slack.MessageEvent) (e slack.RTMEvent) {
	e.Type = "message"
	e.Data = msgEvent

	return e
}

func newMessageEvent(channel string, text string, fromUser string, timestamp string, options ...testMsgOption) (e *slack.MessageEvent) {
	e = new(slack.MessageEvent)
	e.Type = "message"
	e.User = fromUser
	e.Text = text
	e.Timestamp = timestamp
	e.Channel = channel

	for _, applyOption := range options {
		applyOption(e)
	}

	return e
}

func runSlackscotWithIncomingEventsWithLogs(t *testing.T, events []slack.RTMEvent) (sentMessages []sentMessage, updatedMsgs []updatedMessage, deletedMsgs []deletedMessage, logs []string) {
	var logBuilder strings.Builder
	logger := log.New(&logBuilder, "", 0)

	sentMessages, updatedMsgs, deletedMsgs = runSlackscotWithIncomingEvents(t, events, OptionLog(logger))
	return sentMessages, updatedMsgs, deletedMsgs, strings.Split(logBuilder.String(), "\n")
}

func runSlackscotWithIncomingEvents(t *testing.T, events []slack.RTMEvent, option Option) (sentMessages []sentMessage, updatedMsgs []updatedMessage, deletedMsgs []deletedMessage) {
	v := config.NewViperWithDefaults()

	inMemoryChatDriver := inMemoryChatDriver{timeCursor: firstReplyTimestamp - replyTimeIncrementInSeconds, sentMsgs: make([]sentMessage, 0), updatedMsgs: make([]updatedMessage, 0), deletedMsgs: make([]deletedMessage, 0)}
	var infoFinder infoFinder

	s, err := NewSlackscot("chickadee", v, option)
	tp := newTestPlugin()
	s.RegisterPlugin(&tp.Plugin)

	assert.Nil(t, err)

	ec := make(chan slack.RTMEvent)
	termination := make(chan bool)
	go s.handleIncomingEvents(ec, termination, &inMemoryChatDriver, &infoFinder, false)

	go sendTestEventsForProcessing(ec, events)

	<-termination

	return inMemoryChatDriver.sentMsgs, inMemoryChatDriver.updatedMsgs, inMemoryChatDriver.deletedMsgs
}

func sendTestEventsForProcessing(ec chan<- slack.RTMEvent, events []slack.RTMEvent) {
	// Start with a connected event to simulate the normal flow that allows an instance to cache its
	// own identity
	ec <- slack.RTMEvent{Type: "connected_event", Data: &slack.ConnectedEvent{}}

	for _, e := range events {
		ec <- e
	}

	// Terminate the sequence of test events by sending a termination event
	ec <- slack.RTMEvent{Type: "termination", Data: &terminationEvent{}}
}
