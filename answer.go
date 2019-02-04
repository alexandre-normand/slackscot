package slackscot

const (
	ThreadedReplyOpt = "threadedReply"
	BroadcastOpt     = "broadcast"
)

// AnswerOption defines a function applied to Answers
type AnswerOption func(sendOpts map[string]string)

// AnswerInThread sets threaded replying
func AnswerInThread() func(sendOpts map[string]string) {
	return func(sendOpts map[string]string) {
		sendOpts[ThreadedReplyOpt] = "true"
	}
}

// AnswerInThreadWithBroadcast sets threaded replying with broadcast enabled
func AnswerInThreadWithBroadcast() func(sendOpts map[string]string) {
	return func(sendOpts map[string]string) {
		sendOpts[ThreadedReplyOpt] = "true"
		sendOpts[BroadcastOpt] = "true"
	}
}

// AnswerInThreadWithoutBroadcast sets threaded replying with broadcast disabled
func AnswerInThreadWithoutBroadcast() func(sendOpts map[string]string) {
	return func(sendOpts map[string]string) {
		sendOpts[ThreadedReplyOpt] = "true"
		sendOpts[BroadcastOpt] = "false"
	}
}

// AnswerWithoutThreading
func AnswerWithoutThreading() func(sendOpts map[string]string) {
	return func(sendOpts map[string]string) {
		sendOpts[ThreadedReplyOpt] = "false"
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
