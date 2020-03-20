package slackscot

import (
	"github.com/slack-go/slack"
)

// RealTimeMessageSender is implemented by any value that has the NewOutgoingMessage method.
// The main purpose is a slight decoupling of the slack.RTM in order for plugins to be able to write
// tests more easily if all they do is send new messages on a channel
type RealTimeMessageSender interface {
	// NewOutgoingMessage is the function that creates a new message to send. See https://godoc.org/github.com/slack-go/slack#RTM.NewOutgoingMessage for more details
	NewOutgoingMessage(text string, channelID string, options ...slack.RTMsgOption) *slack.OutgoingMessage

	// SendMessage is the function that sends a new real time message. See https://godoc.org/github.com/slack-go/slack#RTM.SendMessage for more details
	SendMessage(outMsg *slack.OutgoingMessage)
}

// messageSender is implemented by any value that has the SendMessage method. Note that the difference between the RealTimeMessageSender
// version is that this one is synchronous and returns the information identifying the sent message.
//
// slack.Client implements this interface
type messageSender interface {
	SendMessage(channelID string, options ...slack.MsgOption) (rChannelID string, rTimestamp string, rText string, err error)
}

// messageUpdater is implemented by any value that has the UpdateMessage method.
//
// slack.Client implements this interface
type messageUpdater interface {
	UpdateMessage(channelID, timestamp string, options ...slack.MsgOption) (rChannelID string, rTimestamp string, rText string, err error)
}

// messageDeleter is implemented by any value that has the DeleteMessage method.
//
// slack.Client implements this interface
type messageDeleter interface {
	DeleteMessage(channelID string, timestamp string) (rChannelID string, rTimestamp string, err error)
}

// ChatDriver encompasses all MessageSender, MessageUpdater and MessageDeleter interfaces and is implemented by any values that
// has all methods of those interfaces
type chatDriver interface {
	messageDeleter
	messageSender
	messageUpdater
}
