package capture

import (
	"github.com/nlopes/slack"
)

// RealTimeSenderCaptor holds messages sent to it keyed
// by channel ID
type RealTimeSenderCaptor struct {
	SentMessages map[string][]string
}

// NewSender returns a new initialized RealTimeSenderCaptor instance
func NewRealTimeSender() (rtms *RealTimeSenderCaptor) {
	rtms = new(RealTimeSenderCaptor)
	rtms.SentMessages = make(map[string][]string)

	return rtms
}

// SendNewMessage captures the details of a sent message (the message itself and the channel it's sent to)
func (rtms *RealTimeSenderCaptor) SendNewMessage(channelID string, message string) (err error) {
	if _, ok := rtms.SentMessages[channelID]; !ok {
		rtms.SentMessages[channelID] = make([]string, 0)
	}

	rtms.SentMessages[channelID] = append(rtms.SentMessages[channelID], message)
	return nil
}

// GetAPI always returns nil
func (rtms *RealTimeSenderCaptor) GetAPI() (rtm *slack.RTM) {
	return nil
}
