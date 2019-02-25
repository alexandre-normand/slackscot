package slackscot

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/schedule"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"
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

type rtmMessage struct {
	channelID string
	message   string
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

type realTimeSender struct {
	rtmMsgs []rtmMessage
}

func (rs *realTimeSender) SendNewMessage(message string, channelID string) (err error) {
	rs.rtmMsgs = append(rs.rtmMsgs, rtmMessage{channelID: channelID, message: message})
	return nil
}

func (rs *realTimeSender) GetAPI() (rtm *slack.RTM) {
	return nil
}

type selfFinder struct {
}

func (i *selfFinder) GetInfo() (user *slack.Info) {
	return &slack.Info{User: &slack.UserDetails{ID: "BotUserID", Name: "Daniel Quinn"}}
}

type userInfoFinder struct {
}

func (u *userInfoFinder) GetUserInfo(userID string) (user *slack.User, err error) {
	return &slack.User{ID: botUserID, Name: "Daniel Quinn"}, nil
}

type emojiReactor struct {
}

func (e *emojiReactor) AddReaction(name string, item slack.ItemRef) error {
	return nil
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

func newTestPlugin() (tp *Plugin) {
	tp = new(Plugin)
	tp.Name = "noRules"
	tp.Commands = []ActionDefinition{{
		Match: func(m *IncomingMessage) bool {
			return strings.HasPrefix(m.NormalizedText, "make")
		},
		Usage:       "make `<something>`",
		Description: "Have the test bot make something for you",
		Answer: func(m *IncomingMessage) *Answer {
			return &Answer{Text: fmt.Sprintf("Make it yourself, @%s", m.User)}
		},
	}}
	tp.HearActions = []ActionDefinition{{
		Hidden: true,
		Match: func(m *IncomingMessage) bool {
			// Only match if the message timestamp matches timestamp1 (the original time). We use this to make sure
			// that slackscot preserves the original message's timestamp when processing message updates
			return m.Msg.Timestamp == timestamp1 && strings.Contains(m.NormalizedText, "blue jays")
		},
		Usage:       "Talk about my secret topic",
		Description: "Reply with usage instructions",
		Answer: func(m *IncomingMessage) *Answer {
			return &Answer{Text: "I heard you say something about blue jays?"}
		},
	}}
	tp.ScheduledActions = nil

	return tp
}

func TestLogfileOverrideUsed(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "test")
	assert.Nil(t, err)

	defer os.Remove(tmpfile.Name()) // clean up

	runSlackscotWithIncomingEvents(t, nil, newTestPlugin(), []slack.RTMEvent{}, OptionLogfile(tmpfile))

	logs, err := ioutil.ReadFile(tmpfile.Name())
	assert.Nil(t, err)

	assert.Contains(t, string(logs), "Connection counter: 0")
}

func TestLatencyReport(t *testing.T) {
	_, _, _, _, logs := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		{Type: "latency_report", Data: &slack.LatencyReport{Value: 120}},
	})

	assert.Contains(t, logs, "Current latency: 120ns")
}

func TestRTMError(t *testing.T) {
	_, _, _, _, logs := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		{Type: "rtm_error", Data: &slack.RTMError{Code: 500, Msg: "test error"}},
	})

	assert.Contains(t, logs, "Error: Code 500 - test error")
}

func TestInvalidCredentialsShutsdownImmediately(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, logs := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		{Type: "invalid_auth_event", Data: &slack.InvalidAuthEvent{}},
		newRTMMessageEvent(newMessageEvent("Cgeneral", "Bonjour", "Alphonse", timestamp1)),
	})

	assert.Contains(t, logs, "Invalid credentials")
	assert.Equal(t, 0, len(sentMsgs))
	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestHandleIncomingMessageTriggeringResponse(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestIgnoreReplyToMessage(t *testing.T) {
	msge := new(slack.MessageEvent)
	msge.Type = "message"
	msge.Channel = "CHGENERAL"
	msge.User = "Alphone"
	msge.Text = "blue jars"
	msge.ReplyTo = 1

	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(msge),
	})

	assert.Equal(t, 0, len(sentMsgs))
	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestIncomingMessageUpdateTriggeringResponseUpdate(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Ignored", timestamp2, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
	}

	if assert.Equal(t, 1, len(updatedMsgs)) {
		assert.Equal(t, 3, len(updatedMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", updatedMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestIncomingMessageUpdateNotTriggeringUpdateIfDifferentChannel(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cother", "blue jays", "Ignored", timestamp2, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	// Check that the messages are distincts and not a message update given they were on different channels
	if assert.Equal(t, 2, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		assert.Equal(t, 3, len(sentMsgs[1].msgOptions))
		assert.Equal(t, "Cother", sentMsgs[1].channelID)
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestThreadedReplies(t *testing.T) {
	v := config.NewViperWithDefaults()
	// Enable threaded replies and disable broadcast
	v.Set(config.ThreadedRepliesKey, true)
	v.Set(config.BroadcastThreadedRepliesKey, false)

	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, v, newTestPlugin(), []slack.RTMEvent{
		// Triggers a new message
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		// Triggers a message update
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Ignored", timestamp2, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		// We can't check for the exact options because they're functions on a non-public nlopes/slack structure but
		// knowing we have 4 options instead of 3 gives some confidence
		assert.Equal(t, 4, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
	}

	if assert.Equal(t, 1, len(updatedMsgs)) {
		assert.Equal(t, 3, len(updatedMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", updatedMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestThreadedRepliesWithBroadcast(t *testing.T) {
	v := config.NewViperWithDefaults()
	// Enable threaded replies and broadcast enabled
	v.Set(config.ThreadedRepliesKey, true)
	v.Set(config.BroadcastThreadedRepliesKey, true)

	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, v, newTestPlugin(), []slack.RTMEvent{
		// Triggers a new message
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		// Triggers a message update
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Ignored", timestamp2, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		// We can't check for the exact options because they're functions on a non-public nlopes/slack structure but
		// knowing we have 5 options instead of 3 gives some confidence that both threaded replies and broadcast are included
		assert.Equal(t, 5, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
	}

	if assert.Equal(t, 1, len(updatedMsgs)) {
		assert.Equal(t, 3, len(updatedMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", updatedMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestIncomingMessageTriggeringNewResponse(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "nothing important", "Alphonse", timestamp1)),
		// This message update should now trigger the hear action
		newRTMMessageEvent(newMessageEvent("Cgeneral", "nothing important", "Ignored", timestamp2, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestIncomingTriggeringMessageUpdatedToNotTriggerAnymore(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp2, optionChangedMessage("never mind", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(updatedMsgs))
	if assert.Equal(t, 1, len(deletedMsgs)) {
		assert.Equal(t, deletedMessage{channelID: "Cgeneral", timestamp: formatTimestamp(firstReplyTimestamp)}, deletedMsgs[0])
		assert.Equal(t, "Cgeneral", deletedMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestDirectMessageMatchingCommand(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		// Trigger the command action
		newRTMMessageEvent(newMessageEvent("DFromUser", "make me happy", "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "DFromUser", sentMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestDirectMessageNotMatchingAnything(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		// Trigger the command action
		newRTMMessageEvent(newMessageEvent("DFromUser", "hey you", "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "DFromUser", sentMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestAtMessageNotMatchingAnything(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		// At Message but not matching the command
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("<@%s> hey you", botUserID), "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestIncomingTriggeringMessageUpdatedToTriggerDifferentAction(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		// Trigger the hear action
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		// Update the message to now trigger the command instead of the hear action
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp2, optionChangedMessage(fmt.Sprintf("<@%s> make me laugh", botUserID), "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 2, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		assert.Equal(t, 3, len(sentMsgs[1].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[1].channelID)
	}

	assert.Equal(t, 0, len(updatedMsgs))

	if assert.Equal(t, 1, len(deletedMsgs)) {
		assert.Equal(t, deletedMessage{channelID: "Cgeneral", timestamp: formatTimestamp(firstReplyTimestamp)}, deletedMsgs[0])
		assert.Equal(t, "Cgeneral", deletedMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

// TestHelpTriggeringNoUserInfoCache indirectly tests the user info caching (or absence of) by exercising the
// help plugin which makes a call to it in order to find info about the user who requested help
func TestHelpTriggeringWithUserInfoCache(t *testing.T) {
	v := config.NewViperWithDefaults()
	v.Set(config.UserInfoCacheSizeKey, 10)

	testHelpTriggering(t, v)
}

func testHelpTriggering(t *testing.T, v *viper.Viper) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, v, newTestPlugin(), []slack.RTMEvent{
		// Trigger the help on a channel
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("<@%s> help", botUserID), "Alphonse", timestamp1)),
		// Trigger the help in a direct message
		newRTMMessageEvent(newMessageEvent("DFromAlphonse", fmt.Sprintf("help"), "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 2, len(sentMsgs)) {
		assert.Equal(t, 5, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		assert.Equal(t, 5, len(sentMsgs[1].msgOptions))
		assert.Equal(t, "DFromAlphonse", sentMsgs[1].channelID)
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

// TestHelpTriggeringNoUserInfoCache indirectly tests the user info caching (or absence of) by exercising the
// help plugin which makes a call to it in order to find info about the user who requested help
func TestHelpTriggeringNoUserInfoCache(t *testing.T) {
	v := config.NewViperWithDefaults()
	v.Set(config.UserInfoCacheSizeKey, 0)

	testHelpTriggering(t, v)
}

func TestTriggeringMessageDeletion(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Ignored", timestamp2, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
	}

	if assert.Equal(t, 1, len(updatedMsgs)) {
		assert.Equal(t, 3, len(updatedMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", updatedMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestIncomingMessageUpdateTriggeringResponseDeletion(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp2, optionDeletedMessage("Cgeneral", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(updatedMsgs))
	if assert.Equal(t, 1, len(deletedMsgs)) {
		assert.Equal(t, deletedMessage{channelID: "Cgeneral", timestamp: formatTimestamp(firstReplyTimestamp)}, deletedMsgs[0])
		assert.Equal(t, "Cgeneral", deletedMsgs[0].channelID)
	}
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestIncomingMessageNotTriggeringResponse(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "bonjour", "Alphonse", timestamp1)),
	})

	assert.Equal(t, 0, len(sentMsgs))
	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestIncomingMessageFromOurselfIgnored(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays are cool", botUserID, timestamp1)),
	})

	assert.Equal(t, 0, len(sentMsgs))
	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.rtmMsgs))
}

func TestScheduledAction(t *testing.T) {
	scheduleDefinition := schedule.Definition{Interval: 1, Unit: schedule.Seconds}
	beatPlugin := Plugin{Name: "rabbit", Commands: nil, HearActions: nil, ScheduledActions: []ScheduledActionDefinition{{Schedule: scheduleDefinition, Description: "Send a beat every second", Action: func(sender RealTimeMessageSender) {
		sender.SendNewMessage("beat", "Cstatus")
	}}}}

	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, &beatPlugin, []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("DFromAlphonse", "help", "Alphonse", timestamp1)),
	})

	// Wait 1.5 seconds so that the first scheduled execution has time to run
	time.Sleep(time.Duration(1500) * time.Millisecond)
	if assert.Equal(t, 1, len(rtmSender.rtmMsgs)) {
		assert.Equal(t, rtmMessage{channelID: "Cstatus", message: "beat"}, rtmSender.rtmMsgs[0])
	}

	assert.Equal(t, 1, len(sentMsgs))
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

func runSlackscotWithIncomingEventsWithLogs(t *testing.T, v *viper.Viper, plugin *Plugin, events []slack.RTMEvent) (sentMessages []sentMessage, updatedMsgs []updatedMessage, deletedMsgs []deletedMessage, rtmSender *realTimeSender, logs []string) {
	var logBuilder strings.Builder
	logger := log.New(&logBuilder, "", 0)

	sentMessages, updatedMsgs, deletedMsgs, rtmSender = runSlackscotWithIncomingEvents(t, v, plugin, events, OptionLog(logger))
	return sentMessages, updatedMsgs, deletedMsgs, rtmSender, strings.Split(logBuilder.String(), "\n")
}

func runSlackscotWithIncomingEvents(t *testing.T, v *viper.Viper, plugin *Plugin, events []slack.RTMEvent, options ...Option) (sentMessages []sentMessage, updatedMsgs []updatedMessage, deletedMsgs []deletedMessage, rtmSender *realTimeSender) {
	if v == nil {
		v = config.NewViperWithDefaults()
	}

	inMemoryChatDriver := inMemoryChatDriver{timeCursor: firstReplyTimestamp - replyTimeIncrementInSeconds, sentMsgs: make([]sentMessage, 0), updatedMsgs: make([]updatedMessage, 0), deletedMsgs: make([]deletedMessage, 0)}
	rtmSender = new(realTimeSender)
	rtmSender.rtmMsgs = make([]rtmMessage, 0)

	var selfFinder selfFinder
	var userInfoFinder userInfoFinder
	var emojiReactor emojiReactor

	s, err := NewSlackscot("chickadee", v, options...)
	s.RegisterPlugin(plugin)

	assert.Nil(t, err)

	timeLoc, err := time.LoadLocation("Local")
	assert.Nil(t, err)

	// Start the scheduler, it is up to the test to wait enough time to make sure scheduled actions run
	go s.startActionScheduler(timeLoc, rtmSender)

	ec := make(chan slack.RTMEvent)
	termination := make(chan bool)
	go s.runInternal(ec, termination, &runDependencies{chatDriver: &inMemoryChatDriver, userInfoFinder: &userInfoFinder, emojiReactor: &emojiReactor, selfInfoFinder: &selfFinder}, false)

	go sendTestEventsForProcessing(ec, events)

	<-termination

	return inMemoryChatDriver.sentMsgs, inMemoryChatDriver.updatedMsgs, inMemoryChatDriver.deletedMsgs, rtmSender
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
