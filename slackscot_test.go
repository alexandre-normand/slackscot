package slackscot

import (
	"encoding/json"
	"fmt"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/schedule"
	"github.com/alexandre-normand/slackscot/test/capture"
	"github.com/gorilla/websocket"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slacktest"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	botUserID                   = "BotUserID"
	formattedBotUserID          = "<@" + botUserID + ">"
	timestamp1                  = "1546833210.036900"
	oneDayLaterTimestamp        = "1546919611.036900" // One second more than 24 hours after timestamp1
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

	endpoint, _, err := slack.UnsafeApplyMsgOptions("", channelID, "", options...)
	if err != nil {
		return "", "", "", err
	}

	respChannelID := channelID
	if strings.Contains(endpoint, "chat.postEphemeral") {
		respChannelID = ""
	}

	return respChannelID, c.nextTimestamp(), fmt.Sprintf("Message on %s", channelID), nil
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

type selfFinder struct {
}

func (i *selfFinder) GetInfo() (user *slack.Info) {
	return &slack.Info{User: &slack.UserDetails{ID: "BotUserID", Name: "Daniel Quinn"}}
}

type userInfoFinder struct {
}

func (u *userInfoFinder) GetUserInfo(userID string) (user *slack.User, err error) {
	return &slack.User{ID: botUserID, Profile: slack.UserProfile{BotID: "b" + botUserID}, RealName: "Daniel Quinn"}, nil
}

type emojiReactor struct {
}

func (e *emojiReactor) AddReaction(name string, item slack.ItemRef) error {
	return nil
}

// Option type for building a message with additional options for specific test cases
type testMsgOption func(e *slack.MessageEvent)

func optionChangedMessage(text string, user string, originalTs string) testMsgOption {
	return func(e *slack.MessageEvent) {
		e.SubType = "message_changed"
		e.SubMessage = &slack.Msg{Text: text, User: user, Timestamp: originalTs}
	}
}

func optionMessageReplied() func(e *slack.MessageEvent) {
	return func(e *slack.MessageEvent) {
		e.SubType = "message_replied"
	}
}

func optionDeletedMessage(channelID string, timestamp string) testMsgOption {
	return func(e *slack.MessageEvent) {
		e.SubType = "message_deleted"
		e.DeletedTimestamp = timestamp
		e.Channel = channelID
	}
}

func optionMessageOnThread(ts string) testMsgOption {
	return func(e *slack.MessageEvent) {
		e.ThreadTimestamp = ts
	}
}

func optionDirectMessage(botUserID string) testMsgOption {
	return func(e *slack.MessageEvent) {
		e.Channel = fmt.Sprintf("D%s", botUserID)
	}
}

func optionBotID(botID string) testMsgOption {
	return func(e *slack.MessageEvent) {
		e.BotID = botID
	}
}

func optionPublicMessageToBot(botUserID string, channelID string) testMsgOption {
	return func(e *slack.MessageEvent) {
		e.Channel = channelID
		e.Text = fmt.Sprintf("<@%s> %s", botUserID, e.Text)
	}
}

func newTestPlugin() (tp *Plugin) {
	tp = new(Plugin)
	tp.Name = "noRules"
	tp.NamespaceCommands = true
	tp.Commands = []ActionDefinition{
		{
			Match: func(m *IncomingMessage) bool {
				return strings.HasPrefix(m.NormalizedText, "make")
			},
			Usage:       "make `<something>`",
			Description: "Have the test bot make something for you",
			Answer: func(m *IncomingMessage) *Answer {
				return &Answer{Text: fmt.Sprintf("Make it yourself, @%s", m.User), Options: []AnswerOption{AnswerEphemeral(m.User)}}
			},
		},
		{
			Match: func(m *IncomingMessage) bool {
				return strings.HasPrefix(m.NormalizedText, "block ")
			},
			Usage:       "block `<something>`",
			Description: "Render your expression as a context block",
			Answer: func(m *IncomingMessage) *Answer {
				expression := strings.TrimPrefix(m.NormalizedText, "block ")
				return &Answer{Text: "", ContentBlocks: []slack.Block{*slack.NewContextBlock("", *slack.NewTextBlockObject("mrkdwn", expression, false, false))}}
			},
		},
		{
			Match: func(m *IncomingMessage) bool {
				return strings.HasPrefix(m.NormalizedText, "create channel ")
			},
			Usage:       "create channel <name>",
			Description: "Creates a new channel with the given name",
			Answer: func(m *IncomingMessage) *Answer {
				expression := strings.TrimPrefix(m.NormalizedText, "create channel ")
				channel, err := tp.SlackClient.CreateChannel(expression)
				if err == nil {
					return &Answer{Text: fmt.Sprintf("Channel #%s created with id: %s", channel.Name, channel.ID)}
				}

				return &Answer{Text: fmt.Sprintf("Error creating channel: %s", err.Error())}
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

func applySlackOptions(opts ...slack.MsgOption) (vals url.Values) {
	_, vals, _ = slack.UnsafeApplyMsgOptions("token", "channel", "url", opts...)
	return vals
}

func TestLogfileOverrideUsed(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "test")
	assert.Nil(t, err)

	defer os.Remove(tmpfile.Name()) // clean up

	runSlackscotWithIncomingEvents(t, nil, newTestPlugin(), []slack.RTMEvent{}, nil, OptionLogfile(tmpfile))

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
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestHandleIncomingMessageTriggeringResponse(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 2, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestAnswerWithNamespacingDisabled(t *testing.T) {
	sentMsgs, _, _, _ := runSlackscotWithIncomingEvents(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("%s make something nice", formattedBotUserID), "Alphonse", timestamp1)),
	}, nil, OptionNoPluginNamespacing())

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "<@Alphonse>: Make it yourself, @Alphonse", vals.Get("text"))
		// It isn't obvious but an ephemeral message is sent *as* the user it's also being sent *to*
		assert.Equal(t, "Alphonse", vals.Get("user"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}
}

func TestAnswerWithContentBlocks(t *testing.T) {
	sentMsgs, _, _, _, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("%s noRules block hello you", formattedBotUserID), "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "<@Alphonse>: ", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
		assert.Equal(t, "[{\"type\":\"context\",\"elements\":{\"Elements\":[{\"type\":\"mrkdwn\",\"text\":\"hello you\"}]}}]", vals.Get("blocks"))
	}
}

func TestAnswerUpdateWithContentBlocks(t *testing.T) {
	sentMsgs, updatedMsgs, _, _, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("%s noRules block hello you", formattedBotUserID), "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("%s noRules block hello you", formattedBotUserID), "Ignored", timestamp2, optionChangedMessage(fmt.Sprintf("%s noRules block hello you and everyone else", formattedBotUserID), "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "<@Alphonse>: ", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
		assert.Equal(t, "[{\"type\":\"context\",\"elements\":{\"Elements\":[{\"type\":\"mrkdwn\",\"text\":\"hello you\"}]}}]", vals.Get("blocks"))
	}

	if assert.Equal(t, 1, len(updatedMsgs)) {
		assert.Equal(t, 3, len(updatedMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", updatedMsgs[0].channelID)

		vals := applySlackOptions(updatedMsgs[0].msgOptions...)
		assert.Equal(t, "<@Alphonse>: ", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
		assert.Equal(t, "[{\"type\":\"context\",\"elements\":{\"Elements\":[{\"type\":\"mrkdwn\",\"text\":\"hello you and everyone else\"}]}}]", vals.Get("blocks"))
	}
}

func TestHandleIncomingThreadedMessageTriggeringResponse(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1, optionMessageOnThread("1212314125"))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
		assert.Equal(t, "1212314125", vals.Get("thread_ts"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestIgnoreIncomingMessageReplied(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1, optionMessageReplied())),
	})

	assert.Empty(t, sentMsgs)
	assert.Empty(t, updatedMsgs)
	assert.Empty(t, deletedMsgs)
	assert.Empty(t, rtmSender.SentMessages)
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
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestIncomingMessageUpdateTriggeringResponseUpdate(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Ignored", timestamp2, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 2, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	if assert.Equal(t, 1, len(updatedMsgs)) {
		assert.Equal(t, 2, len(updatedMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", updatedMsgs[0].channelID)

		vals := applySlackOptions(updatedMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestIncomingMessageUpdateNotTriggeringUpdateIfDifferentChannel(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cother", "blue jays", "Ignored", timestamp2, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	// Check that the messages are distincts and not a message update given they were on different channels
	if assert.Equal(t, 2, len(sentMsgs)) {
		assert.Equal(t, 2, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))

		assert.Equal(t, 2, len(sentMsgs[1].msgOptions))
		assert.Equal(t, "Cother", sentMsgs[1].channelID)
		vals = applySlackOptions(sentMsgs[1].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
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
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
		assert.Equal(t, timestamp1, vals.Get("thread_ts"))
	}

	if assert.Equal(t, 1, len(updatedMsgs)) {
		assert.Equal(t, 2, len(updatedMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", updatedMsgs[0].channelID)

		vals := applySlackOptions(updatedMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
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
		assert.Equal(t, 4, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
		assert.Equal(t, timestamp1, vals.Get("thread_ts"))
		assert.Equal(t, "true", vals.Get("reply_broadcast"))
	}

	if assert.Equal(t, 1, len(updatedMsgs)) {
		assert.Equal(t, 2, len(updatedMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", updatedMsgs[0].channelID)

		vals := applySlackOptions(updatedMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestIncomingMessageTriggeringNewResponse(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "nothing important", "Alphonse", timestamp1)),
		// This message update should now trigger the hear action
		newRTMMessageEvent(newMessageEvent("Cgeneral", "nothing important", "Ignored", timestamp2, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 2, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestIncomingTriggeringMessageUpdatedToNotTriggerAnymore(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp2, optionChangedMessage("never mind", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 2, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(updatedMsgs))

	if assert.Equal(t, 1, len(deletedMsgs)) {
		assert.Equal(t, deletedMessage{channelID: "Cgeneral", timestamp: formatTimestamp(firstReplyTimestamp)}, deletedMsgs[0])
		assert.Equal(t, "Cgeneral", deletedMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestDirectMessageMatchingCommand(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		// Trigger the command action
		newRTMMessageEvent(newMessageEvent("DFromUser", "noRules make me happy", "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "DFromUser", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "Make it yourself, @Alphonse", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
		assert.Equal(t, "", vals.Get("thread_ts"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestDirectMessageNotMatchingAnything(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		// Trigger the command action
		newRTMMessageEvent(newMessageEvent("DFromUser", "hey you", "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 2, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "DFromUser", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I don't understand. Ask me for \"help\" to get a list of things I do", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
		assert.Equal(t, "", vals.Get("thread_ts"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestDefaultCommandAnswerInChannel(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		// Trigger the command action
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("%s mistyped command", formattedBotUserID), "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 2, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "<@Alphonse>: I don't understand. Ask me for \"help\" to get a list of things I do", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestDefaultCommandAnswerToMsgOnExistingThread(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		// Trigger the command action
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("%s mistyped command", formattedBotUserID), "Alphonse", timestamp1, optionMessageOnThread("1212314125"))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "<@Alphonse>: I don't understand. Ask me for \"help\" to get a list of things I do", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
		assert.Equal(t, "1212314125", vals.Get("thread_ts"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestAtMessageNotMatchingAnything(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		// At Message but not matching the command
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("%s hey you", formattedBotUserID), "Alphonse", timestamp1)),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 2, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "<@Alphonse>: I don't understand. Ask me for \"help\" to get a list of things I do", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestIncomingTriggeringMessageUpdatedToTriggerDifferentAction(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		// Trigger the hear action
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		// Update the message to now trigger the command instead of the hear action
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp2, optionChangedMessage(fmt.Sprintf("<@%s> noRules make me laugh", botUserID), "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 2, len(sentMsgs)) {
		assert.Equal(t, 2, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))

		assert.Equal(t, 3, len(sentMsgs[1].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[1].channelID)
		vals = applySlackOptions(sentMsgs[1].msgOptions...)
		assert.Equal(t, "<@Alphonse>: Make it yourself, @Alphonse", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(updatedMsgs))

	if assert.Equal(t, 1, len(deletedMsgs)) {
		assert.Equal(t, deletedMessage{channelID: "Cgeneral", timestamp: formatTimestamp(firstReplyTimestamp)}, deletedMsgs[0])
		assert.Equal(t, "Cgeneral", deletedMsgs[0].channelID)
	}

	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

// Test that we send a new message when the previous answer is not updatable because it was ephemeral
func TestMessageUpdateNoUpdateToEphemeralAnswer(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		// Trigger the original answer that is sent as an ephemeral message
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("<@%s> noRules make me laugh", botUserID), "Alphonse", timestamp1)),
		// Update the message to change the message slightly
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("<@%s> noRules make me cry", botUserID), "Alphonse", timestamp2, optionChangedMessage(fmt.Sprintf("<@%s> noRules make me cry", botUserID), "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 2, len(sentMsgs)) {
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "<@Alphonse>: Make it yourself, @Alphonse", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))

		assert.Equal(t, 3, len(sentMsgs[1].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[1].channelID)
		vals = applySlackOptions(sentMsgs[1].msgOptions...)
		assert.Equal(t, "<@Alphonse>: Make it yourself, @Alphonse", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

// TestHelpTriggeringWithUserInfoCache indirectly tests the user info caching (or absence of) by exercising the
// help plugin which makes a call to it in order to find info about the user who requested help
func TestHelpTriggeringWithUserInfoCache(t *testing.T) {
	v := config.NewViperWithDefaults()
	v.Set(config.UserInfoCacheSizeKey, 10)
	v.Set(config.MessageProcessingPartitionCount, 1)

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
		assert.Equal(t, 3, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, fmt.Sprintf("<@Alphonse>: 🤝 Hi, `Daniel Quinn`! I'm `chickadee` (engine `v%s`) and I listen to the team's "+
			"chat and provides automated functions :genie:.\n\nI currently support the following commands:\n\t• `noRules make `<something>`` - "+
			"Have the test bot make something for you\n\t• `noRules block `<something>`` - Render your expression as a context block\n"+
			"\t• `noRules create channel <name>` - Creates a new channel with the given name\n", VERSION), vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
		assert.Equal(t, timestamp1, vals.Get("thread_ts"))

		assert.Equal(t, 2, len(sentMsgs[1].msgOptions))
		assert.Equal(t, "DFromAlphonse", sentMsgs[1].channelID)
		vals = applySlackOptions(sentMsgs[1].msgOptions...)
		assert.Equal(t, fmt.Sprintf("🤝 Hi, `Daniel Quinn`! I'm `chickadee` (engine `v%s`) and I listen to the team's "+
			"chat and provides automated functions :genie:.\n\nI currently support the following commands:\n\t• `noRules make `<something>`` - "+
			"Have the test bot make something for you\n\t• `noRules block `<something>`` - Render your expression as a context block\n"+
			"\t• `noRules create channel <name>` - Creates a new channel with the given name\n", VERSION), vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

// TestHelpTriggeringNoUserInfoCache indirectly tests the user info caching (or absence of) by exercising the
// help plugin which makes a call to it in order to find info about the user who requested help
func TestHelpTriggeringNoUserInfoCache(t *testing.T) {
	v := config.NewViperWithDefaults()
	v.Set(config.UserInfoCacheSizeKey, 0)
	v.Set(config.MessageProcessingPartitionCount, 1)

	testHelpTriggering(t, v)
}

func TestIncomingMessageUpdateTriggeringResponseDeletion(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp2, optionDeletedMessage("Cgeneral", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, 2, len(sentMsgs[0].msgOptions))
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)
		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	if assert.Equal(t, 1, len(deletedMsgs)) {
		assert.Equal(t, deletedMessage{channelID: "Cgeneral", timestamp: formatTimestamp(firstReplyTimestamp)}, deletedMsgs[0])
		assert.Equal(t, "Cgeneral", deletedMsgs[0].channelID)
	}
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestIncomingMessageNotTriggeringResponse(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "bonjour", "Alphonse", timestamp1)),
	})

	assert.Equal(t, 0, len(sentMsgs))
	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestIncomingMessageFromOurselfIgnored(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays are cool", botUserID, timestamp1)),
	})

	assert.Equal(t, 0, len(sentMsgs))
	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestIncomingMessageFromOurselfWithBotIDIgnored(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays are cool", "", timestamp1, optionBotID("b"+botUserID))),
	})

	assert.Equal(t, 0, len(sentMsgs))
	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestScheduledAction(t *testing.T) {
	scheduleDefinition := schedule.Definition{Interval: 1, Unit: schedule.Seconds}
	beatPlugin := new(Plugin)
	beatPlugin.Name = "beat"
	beatPlugin.ScheduledActions = []ScheduledActionDefinition{{Schedule: scheduleDefinition, Description: "Send a beat every second", Action: func() {
		om := beatPlugin.RealTimeMsgSender.NewOutgoingMessage("beat", "Cstatus")
		beatPlugin.RealTimeMsgSender.SendMessage(om)
	}}}

	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, beatPlugin, []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("DFromAlphonse", "help", "Alphonse", timestamp1)),
	})

	// Wait 1.5 seconds so that the first scheduled execution has time to run
	time.Sleep(time.Duration(1500) * time.Millisecond)
	if assert.Equal(t, 1, len(rtmSender.SentMessages)) {
		assert.Contains(t, rtmSender.SentMessages, "Cstatus")
		assert.Contains(t, rtmSender.SentMessages["Cstatus"], "beat")
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

func TestMessageUpdatedAfterHandlingThresholdIgnored(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Ignored", oneDayLaterTimestamp, optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))
		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

// This shouldn't happen but if slack was sending invalid message timestamps (not float values), we
// want to default to handling the message
func TestMessageUpdatedHandledWhenUnableToCalculateAge(t *testing.T) {
	sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", "blue jays", "Alphonse", timestamp1)),
		// First case where we set the updated message to an invalid original time
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("<@%s> noRules make me something nice", botUserID), "Alphonse", oneDayLaterTimestamp, optionChangedMessage("blue jays eat acorn", "Alphonse", "invalid"))),
		// Second case where we set the updated message to an invalid new message time
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("<@%s> noRules make me cry", botUserID), "Alphonse", "not a float", optionChangedMessage("blue jays eat acorn", "Alphonse", timestamp1))),
	})

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "I heard you say something about blue jays?", vals.Get("text"))

		assert.Equal(t, "true", vals.Get("as_user"))
	}

	assert.Equal(t, 0, len(updatedMsgs))
	assert.Equal(t, 0, len(deletedMsgs))
	assert.Equal(t, 0, len(rtmSender.SentMessages))
}

func TestOptionWithSlackOptionApplied(t *testing.T) {
	testServer := slacktest.NewTestServer()
	testServer.Handle("/channels.create", slacktest.Websocket(func(conn *websocket.Conn) {
		if err := slacktest.RTMServerSendGoodbye(conn); err != nil {
			log.Println("failed to send goodbye", err)
		}
	}))

	testServer.Start()
	defer testServer.Stop()

	termination := make(chan bool)
	s, err := New("BobbyTables", config.NewViperWithDefaults(), OptionWithSlackOption(slack.OptionAPIURL(testServer.GetAPIURL())), OptionTestMode(termination))
	require.NoError(t, err)

	tp := newTestPlugin()
	s.RegisterPlugin(tp)

	go s.Run()

	testStart := time.Now()
	for now := time.Now(); tp.SlackClient == nil && now.Sub(testStart) < time.Duration(1)*time.Second; now = time.Now() {
		time.Sleep(10 * time.Millisecond)
	}
	require.NotNil(t, tp.SlackClient)

	testServer.SendToWebsocket("{\"type\":\"goodbye\"}")
	// Wait for slackscot to terminate
	<-termination
}

func TestSlackClientUsageFromPlugin(t *testing.T) {
	createResponse := `{
	    "ok": true,
	    "channel": {
	        "id": "CRANDOM",
	        "name": "offgrid",
	        "is_channel": true,
	        "created": 1576036682,
	        "is_archived": false,
	        "is_general": false,
	        "unlinked": 0,
	        "creator": "USER",
	        "name_normalized": "offgrid",
	        "is_shared": false,
	        "is_org_shared": false,
	        "is_member": true,
	        "is_private": false,
	        "is_mpim": false,
	        "last_read": "0000000000.000000",
	        "latest": null,
	        "unread_count": 0,
	        "unread_count_display": 0,
	        "members": [
	            "USER"
	        ],
	        "topic": {
	            "value": "",
	            "creator": "",
	            "last_set": 0
	        },
	        "purpose": {
	            "value": "",
	            "creator": "",
	            "last_set": 0
	        },
	        "previous_names": [],
	        "priority": 0
	    }
	}`

	handler := func(c slacktest.Customize) {
		c.Handle("/channels.create", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(createResponse))
		})
	}

	testServer := slacktest.NewTestServer(handler)
	testServer.Start()
	defer testServer.Stop()

	sentMsgs, _, _, _ := runSlackscotWithIncomingEvents(t, nil, newTestPlugin(), []slack.RTMEvent{
		newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("%s noRules create channel #offgrid", formattedBotUserID), "Alphonse", timestamp1)),
	}, testServer)

	if assert.Equal(t, 1, len(sentMsgs)) {
		assert.Equal(t, "Cgeneral", sentMsgs[0].channelID)

		vals := applySlackOptions(sentMsgs[0].msgOptions...)
		assert.Equal(t, "<@Alphonse>: Channel #offgrid created with id: CRANDOM", vals.Get("text"))
	}
}

func TestPartitionCountConfigurations(t *testing.T) {
	tests := map[string]struct {
		partitionCount int
		expectedError  string
	}{
		"InvalidZeroPartitions": {
			partitionCount: 0,
			expectedError:  "advanced.messageProcessingPartitionCount config should be a power of two but was [0]",
		},
		"ValidOnePartition": {
			partitionCount: 1,
			expectedError:  "",
		},
		"ValidTwoPartitions": {
			partitionCount: 2,
			expectedError:  "",
		},
		"Invalid3Partitions": {
			partitionCount: 3,
			expectedError:  "advanced.messageProcessingPartitionCount config should be a power of two but was [3]",
		},
		"Valid4Partitions": {
			partitionCount: 4,
			expectedError:  "",
		},
		"Invalid5Partitions": {
			partitionCount: 5,
			expectedError:  "advanced.messageProcessingPartitionCount config should be a power of two but was [5]",
		},
		"Valid8Partitions": {
			partitionCount: 8,
			expectedError:  "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			v := config.NewViperWithDefaults()
			v.Set(config.MessageProcessingPartitionCount, tc.partitionCount)

			_, err := New("chickadee", v)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

// This test validates the important part of concurrent message processing.
// It achieves this by setting up a plugin with a command that waits for a signal that is
// emitted by another command. It then uses this feature
// 1. Trigger the first command (which waits on a message sent for the second command)
// 2. Send an update on that same message (to be queued in the same partition to be processed after the original message)
// 3. Trigger the second command.
// If the concurrent processing is working as intended, the 3rd message is processed concurrently and sends the signal to open
// the gate and unblock processing on the first partition on which the first message and its update are waiting.
//
// The validation can't validate the ordering of the first two messages as the concurrent processing of messages on different
// partitions isn't guaranteed but it does ensure that the message update to the first message is processed after by looking at
// the updated message's sequence number.
func TestConcurrentProcessingOfNonRelatedMessages(t *testing.T) {
	testComplete := make(chan bool)

	go func() {
		orderCount := 0
		startMaker := make(chan bool)
		makerStarted := false

		tp := new(Plugin)
		tp.Name = "maker"
		tp.NamespaceCommands = false
		tp.Commands = []ActionDefinition{
			{
				Match: func(m *IncomingMessage) bool {
					return strings.HasPrefix(m.NormalizedText, "wait and make")
				},
				Usage:       "wait and make <something>",
				Description: "Simulation of a long running command waiting for external IO",
				Answer: func(m *IncomingMessage) *Answer {
					if !makerStarted {
						<-startMaker
						makerStarted = true
					}

					orderCount++
					return &Answer{Text: fmt.Sprintf("Order #%d made %s", orderCount, strings.TrimPrefix(m.NormalizedText, "wait and make "))}
				},
			},
			{
				Match: func(m *IncomingMessage) bool {
					return strings.HasPrefix(m.NormalizedText, "just make")
				},
				Usage:       "just make <something>",
				Description: "",
				Answer: func(m *IncomingMessage) *Answer {
					orderCount++
					a := &Answer{Text: fmt.Sprintf("Order #%d made %s immediately", orderCount, strings.TrimPrefix(m.NormalizedText, "just make "))}
					if !makerStarted {
						startMaker <- true
					}

					return a
				},
			},
		}

		v := config.NewViperWithDefaults()
		v.Set(config.MessageProcessingPartitionCount, 2)

		sentMsgs, updatedMsgs, deletedMsgs, rtmSender, _ := runSlackscotWithIncomingEventsWithLogs(t, v, tp, []slack.RTMEvent{
			newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("%s wait and make cake", formattedBotUserID), "Alphonse", timestamp1)),
			newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("%s wait and make cake", formattedBotUserID), "Ignored", "1992929", optionChangedMessage(fmt.Sprintf("%s wait and make big cake", formattedBotUserID), "Alphonse", timestamp1))),
			newRTMMessageEvent(newMessageEvent("Cgeneral", fmt.Sprintf("%s just make juice", formattedBotUserID), "Alphonso", "3253298")),
		})

		if assert.Equal(t, 2, len(sentMsgs)) {
			msgs := make([]string, 0)
			// We want to ignore the ordering of messages that aren't related and don't share the same parent message
			for _, msg := range sentMsgs {
				vals := applySlackOptions(msg.msgOptions...)
				msgs = append(msgs, vals.Get("text"))
			}

			assert.Contains(t, msgs, "<@Alphonso>: Order #1 made juice immediately")
			assert.Contains(t, msgs, "<@Alphonse>: Order #2 made cake")
		}

		if assert.Equal(t, 1, len(updatedMsgs)) {
			vals := applySlackOptions(updatedMsgs[0].msgOptions...)
			assert.Equal(t, "<@Alphonse>: Order #3 made big cake", vals.Get("text"))
		}

		assert.Equal(t, 0, len(deletedMsgs))
		assert.Equal(t, 0, len(rtmSender.SentMessages))

		testComplete <- true
	}()

	// Make sure the test runs under a second. It should actually run very quickly so having this timeout
	// help reports the failure with a friendlier message than a very long higher level timeout or leaving
	// the user waiting after a test hanging
	select {
	case <-testComplete:
		// Test ran successfully
	case <-time.After(1 * time.Second):
		t.Error("Failed to run test in 1 second. Did you break concurrent message processing?")
	}
}

func TestSlackMessageIDStringer(t *testing.T) {
	assert.Equal(t, "channel/2324", SlackMessageID{"channel", "2324"}.String())
}

func newRTMMessageEvent(msgEvent *slack.MessageEvent) (e slack.RTMEvent) {
	e.Type = "message"
	e.Data = msgEvent

	return e
}

func msgToJson(e slack.RTMEvent) (val string) {
	rendered, err := json.Marshal(e)
	if err != nil {
		return ""
	}

	return string(rendered)
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

func runSlackscotWithIncomingEventsWithLogs(t *testing.T, v *viper.Viper, plugin *Plugin, events []slack.RTMEvent) (sentMessages []sentMessage, updatedMsgs []updatedMessage, deletedMsgs []deletedMessage, rtmSenderCaptor *capture.RealTimeSenderCaptor, logs []string) {
	var logBuilder strings.Builder
	logger := log.New(&logBuilder, "", 0)

	sentMessages, updatedMsgs, deletedMsgs, rtmSenderCaptor = runSlackscotWithIncomingEvents(t, v, plugin, events, nil, OptionLog(logger))
	return sentMessages, updatedMsgs, deletedMsgs, rtmSenderCaptor, strings.Split(logBuilder.String(), "\n")
}

func runSlackscotWithIncomingEvents(t *testing.T, v *viper.Viper, plugin *Plugin, events []slack.RTMEvent, slackTestServer *slacktest.Server, options ...Option) (sentMessages []sentMessage, updatedMsgs []updatedMessage, deletedMsgs []deletedMessage, rtmSenderCaptor *capture.RealTimeSenderCaptor) {
	if v == nil {
		v = config.NewViperWithDefaults()
		v.Set(config.MessageProcessingPartitionCount, 1)
	}

	inMemoryChatDriver := inMemoryChatDriver{timeCursor: firstReplyTimestamp - replyTimeIncrementInSeconds, sentMsgs: make([]sentMessage, 0), updatedMsgs: make([]updatedMessage, 0), deletedMsgs: make([]deletedMessage, 0)}
	rtmSenderCaptor = capture.NewRealTimeSender()

	var selfFinder selfFinder
	var userInfoFinder userInfoFinder
	var emojiReactor emojiReactor

	tcm := NewTestCmdMatcher(formattedBotUserID + " ")

	termination := make(chan bool)
	options = append(options, OptionTestMode(termination))
	s, err := New("chickadee", v, options...)

	s.cmdMatcher = tcm
	//s.botMatcher = tcm

	s.RegisterPlugin(plugin)

	assert.Nil(t, err)

	timeLoc, err := time.LoadLocation("Local")
	assert.Nil(t, err)

	// Start the scheduler, it is up to the test to wait enough time to make sure scheduled actions run
	go s.startActionScheduler(timeLoc)

	ec := make(chan slack.RTMEvent)

	var sc *slack.Client
	if slackTestServer != nil {
		sc = slack.New("", slack.OptionAPIURL(slackTestServer.GetAPIURL()))
		require.NotNil(t, sc)
	}

	go s.runInternal(ec, &runDependencies{chatDriver: &inMemoryChatDriver, userInfoFinder: &userInfoFinder, emojiReactor: &emojiReactor, selfInfoFinder: &selfFinder, realTimeMsgSender: rtmSenderCaptor, slackClient: sc})

	go sendTestEventsForProcessing(ec, events)

	<-termination

	s.Close()
	return inMemoryChatDriver.sentMsgs, inMemoryChatDriver.updatedMsgs, inMemoryChatDriver.deletedMsgs, rtmSenderCaptor
}

func sendTestEventsForProcessing(ec chan<- slack.RTMEvent, events []slack.RTMEvent) {
	// Start with a connected event to simulate the normal flow that allows an instance to cache its
	// own identity
	ec <- slack.RTMEvent{Type: "connected_event", Data: &slack.ConnectedEvent{}}

	for _, e := range events {
		ec <- e
	}

	// Terminate the sequence of test events by sending a termination event
	ec <- slack.RTMEvent{"disconnected", &slack.DisconnectedEvent{Intentional: true, Cause: slack.ErrRTMGoodbye}}
}
