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
	c.timeCursor = c.timeCursor + 10
	return fmt.Sprintf("%d.000", c.timeCursor)
}

type infoFinder struct {
}

type userInfoFinder struct {
}

type testPlugin struct {
	Plugin
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
	return &slack.User{ID: "BotUserID", Name: "Daniel Quinn"}, nil
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

func TestInvalidCredentialsShutsdownImmediately(t *testing.T) {
	sentMsgs, logs := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		slack.RTMEvent{Type: "invalid_auth_event", Data: &slack.InvalidAuthEvent{}},
		newRTMMessageEvent(newMessageEvent("Bonjour", "Alphonse")),
	})

	assert.Contains(t, logs, "Invalid credentials")
	assert.Equal(t, 0, len(sentMsgs))
}

func TestHandleIncomingMessageTriggeringResponse(t *testing.T) {
	sentMsgs, _ := runSlackscotWithIncomingEventsWithLogs(t, []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Bonjour", "Alphonse")),
		newRTMMessageEvent(newMessageEvent("blue jays", "Alphonse")),
	})

	assert.Equal(t, 1, len(sentMsgs))
	assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
}

func newRTMMessageEvent(msgEvent *slack.MessageEvent) (e slack.RTMEvent) {
	e.Type = "message"
	e.Data = msgEvent

	return e
}

func newMessageEvent(text string, user string) (msge *slack.MessageEvent) {
	msge = new(slack.MessageEvent)
	msge.Type = "message"
	msge.Channel = "CHGENERAL"
	msge.User = user
	msge.Text = text

	return msge
}

func runSlackscotWithIncomingEventsWithLogs(t *testing.T, events []slack.RTMEvent) (sentMessages []sentMessage, logs []string) {
	var logBuilder strings.Builder
	logger := log.New(&logBuilder, "", 0)

	return runSlackscotWithIncomingEvents(t, events, OptionLog(logger)), strings.Split(logBuilder.String(), "\n")
}

func runSlackscotWithIncomingEvents(t *testing.T, events []slack.RTMEvent, option Option) (sentMessages []sentMessage) {
	v := config.NewViperWithDefaults()

	inMemoryChatDriver := inMemoryChatDriver{timeCursor: 1547785956, sentMsgs: make([]sentMessage, 0), updatedMsgs: make([]updatedMessage, 0), deletedMsgs: make([]deletedMessage, 0)}
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

	return inMemoryChatDriver.sentMsgs
}

func sendTestEventsForProcessing(ec chan<- slack.RTMEvent, events []slack.RTMEvent) {
	// Start with a connected event to simulate the normal flow that allows an instance to cache its
	// own identity
	ec <- slack.RTMEvent{Type: "connected_event", Data: &slack.ConnectedEvent{}}

	for _, e := range events {
		log.Printf("Sending event %v\n", e)
		ec <- e
	}

	log.Printf("Sending termination event\n")
	// Terminate the sequence of test events by sending a termination event
	ec <- slack.RTMEvent{Type: "termination", Data: &terminationEvent{}}
}
