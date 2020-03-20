package slackscot

import (
	"github.com/slack-go/slack"
)

const (
	// ThreadedReplyOpt is the name of the option indicating a threaded-reply answer
	ThreadedReplyOpt = "threadedReply"
	// BroadcastOpt is the name of the option indicating a broadcast answer
	BroadcastOpt = "broadcast"
	// ThreadTimestamp is the name of the option indicating the explicit timestamp of the thread to reply to
	ThreadTimestamp = "threadTimestamp"
	// EphemeralAnswerToOpt marks an answer to be sent as an ephemeral message to the provided userID
	EphemeralAnswerToOpt = "ephemeralMsgToUserID"
)

// Answer holds data of an Action's Answer: namely, its text and options
// to use when delivering it
type Answer struct {
	Text string

	// Options to apply when sending a message
	Options []AnswerOption

	// BlockKit content blocks to apply when sending the message
	ContentBlocks []slack.Block
}

// AnswerOption defines a function applied to Answers
type AnswerOption func(sendOpts map[string]string)

// AnswerInThread sets threaded replying
func AnswerInThread() AnswerOption {
	return func(sendOpts map[string]string) {
		sendOpts[ThreadedReplyOpt] = "true"
	}
}

// AnswerInExistingThread sets threaded replying with the existing thread timestamp
func AnswerInExistingThread(threadTimestamp string) AnswerOption {
	return func(sendOpts map[string]string) {
		sendOpts[ThreadedReplyOpt] = "true"
		sendOpts[ThreadTimestamp] = threadTimestamp
	}
}

// AnswerInThreadWithBroadcast sets threaded replying with broadcast enabled
func AnswerInThreadWithBroadcast() AnswerOption {
	return func(sendOpts map[string]string) {
		sendOpts[ThreadedReplyOpt] = "true"
		sendOpts[BroadcastOpt] = "true"
	}
}

// AnswerInThreadWithoutBroadcast sets threaded replying with broadcast disabled
func AnswerInThreadWithoutBroadcast() AnswerOption {
	return func(sendOpts map[string]string) {
		sendOpts[ThreadedReplyOpt] = "true"
		sendOpts[BroadcastOpt] = "false"
	}
}

// AnswerWithoutThreading sets an answer to threading (and implicitly, broadcast) disabled
func AnswerWithoutThreading() AnswerOption {
	return func(sendOpts map[string]string) {
		sendOpts[ThreadedReplyOpt] = "false"
	}
}

// AnswerEphemeral sends the answer as an ephemeral message to the provided userID
func AnswerEphemeral(userID string) AnswerOption {
	return func(sendOpts map[string]string) {
		sendOpts[EphemeralAnswerToOpt] = userID
	}
}

// ApplyAnswerOpts applies answering options to build the send configuration
func ApplyAnswerOpts(opts ...AnswerOption) (sendOptions map[string]string) {
	sendOptions = make(map[string]string)
	for _, opt := range opts {
		opt(sendOptions)
	}

	return sendOptions
}
