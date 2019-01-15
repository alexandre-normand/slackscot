package slackscot

import (
	"github.com/nlopes/slack"
)

// RealTimeMessageSender is implemented by any value that has the SendNewMessage and GetAPI method.
// The main purpose is a slight decoupling of the slack.RTM in order for plugins to be able to write
// tests more easily if all they do is send new messages on a channel. GetAPI leaks the slack.RTM
// for more advanced uses.
type RealTimeMessageSender interface {
	// SendNewMessage is the function that sends a new message to the specified channelId
	SendNewMessage(message string, channelId string) (err error)

	// GetAPI is a function that returns the internal slack RTM
	GetAPI() *slack.RTM
}

// messageSender is implemented by any value that has the SendMessage method. Note that the difference between the RealTimeMessageSender
// version is that this one is synchronous and returns the information identifying the sent message.
//
// slack.Client implements this interface
type messageSender interface {
	SendMessage(channelId string, options ...slack.MsgOption) (rChannelId string, rTimestamp string, rText string, err error)
}

// messageCreator is implemented by any value that has the NewOutgoingMessage method. This is the preferred way to create a new outgoing message
// as it sets the message identifier.
// TODO: remove this unnecessary usage since we ignore the id it generates when sending
type messageCreator interface {
	NewOutgoingMessage(text string, channelId string, options ...slack.RTMsgOption) *slack.OutgoingMessage
}

// messageCreatorSender is implemented by any value that has the messageCreator and messageSender interfaces
type messageCreatorSender interface {
	messageCreator
	messageSender
}

// messageUpdater is implemented by any value that has the UpdateMessage method.
//
// slack.Client implements this interface
type messageUpdater interface {
	UpdateMessage(channelId, timestamp string, options ...slack.MsgOption) (rChannelId string, rTimestamp string, rText string, err error)
}

// messageDeleter is implemented by any value that has the DeleteMessage method.
//
// slack.Client implements this interface
type messageDeleter interface {
	DeleteMessage(channelId string, timestamp string) (rChannelId string, rTimestamp string, err error)
}

// ChatDriver encompasses all MessageSender, MessageUpdater and MessageDeleter interfaces and is implemented by any values that
// has all methods of those interfaces
type chatDriver interface {
	messageDeleter
	messageSender
	messageUpdater
}

// slackRealTimeMsgSender is the default and main implementing type for the AdvancedMessageSender interface
type slackRealTimeMsgSender struct {
	rtm *slack.RTM
}

// SendNewMessage sends a new message using the slack RTM api
func (s *slackRealTimeMsgSender) SendNewMessage(message string, channelId string) (err error) {
	m := s.rtm.NewOutgoingMessage(message, channelId)
	s.rtm.SendMessage(m)

	return nil
}

// GetAPI returns the underlying slack RTM api. Beware that relying on this when writing a plugin
// might mean complications in writing tests for it
func (s *slackRealTimeMsgSender) GetAPI() *slack.RTM {
	return s.rtm
}
