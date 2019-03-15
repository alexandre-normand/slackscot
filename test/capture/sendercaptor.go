package capture

import (
	"github.com/nlopes/slack"
)

// RealTimeSenderCaptor holds messages sent to it keyed
// by channel ID
type RealTimeSenderCaptor struct {
	SentMessages map[string][]string
}

// NewRealTimeSender returns a new initialized RealTimeSenderCaptor instance
func NewRealTimeSender() (rtms *RealTimeSenderCaptor) {
	rtms = new(RealTimeSenderCaptor)
	rtms.SentMessages = make(map[string][]string)

	return rtms
}

// NewOutgoingMessage captures the details of a sent message (the message itself and the channel it's sent to)
// The returned OutgoingMessage has only the channel ID and text set on it
func (rtms *RealTimeSenderCaptor) NewOutgoingMessage(text string, channelID string, options ...slack.RTMsgOption) *slack.OutgoingMessage {
	if _, ok := rtms.SentMessages[channelID]; !ok {
		rtms.SentMessages[channelID] = make([]string, 0)
	}

	rtms.SentMessages[channelID] = append(rtms.SentMessages[channelID], text)
	return &slack.OutgoingMessage{Channel: channelID, Text: text}
}
