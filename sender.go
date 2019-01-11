package slackscot

import (
	"github.com/nlopes/slack"
)

// MessageSender is implemented by any value that has the SendNewMessage and GetAPI method.
// The main purpose is a slight decoupling of the slack.RTM in order for plugins to be able to write
// tests more easily if all they do is send new messages on a channel. GetAPI leaks the slack.RTM
// for more advanced uses.
type MessageSender interface {
	// SendNewMessage is the function that sends a new message to the specified channelId
	SendNewMessage(message string, channelId string) (err error)

	// GetAPI is a function that returns the internal slack RTM
	GetAPI() *slack.RTM
}

// slackMsgSender is the default and main implementing type for the MessageSender interface
type slackMsgSender struct {
	rtm *slack.RTM
}

// SendNewMessage sends a new message using the slack RTM api
func (s *slackMsgSender) SendNewMessage(message string, channelId string) (err error) {
	m := s.rtm.NewOutgoingMessage(message, channelId)
	s.rtm.SendMessage(m)

	return nil
}

// GetAPI returns the underlying slack RTM api. Beware that relying on this when writing a plugin
// might mean complications in writing tests for it
func (s *slackMsgSender) GetAPI() *slack.RTM {
	return s.rtm
}
