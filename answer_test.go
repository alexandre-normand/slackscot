package slackscot_test

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestApplyAnswerOptions(t *testing.T) {
	testCases := []struct {
		name           string
		options        []slackscot.AnswerOption
		expectedConfig map[string]string
	}{
		{"none", []slackscot.AnswerOption{}, make(map[string]string)},
		{"threadedReply", []slackscot.AnswerOption{slackscot.AnswerInThread()}, map[string]string{slackscot.ThreadedReplyOpt: "true"}},
		{"threadedReplyWithBroadcast", []slackscot.AnswerOption{slackscot.AnswerInThreadWithBroadcast()}, map[string]string{slackscot.ThreadedReplyOpt: "true", slackscot.BroadcastOpt: "true"}},
		{"threadedReplyWithoutBroadcast", []slackscot.AnswerOption{slackscot.AnswerInThreadWithoutBroadcast()}, map[string]string{slackscot.ThreadedReplyOpt: "true", slackscot.BroadcastOpt: "false"}},
		{"noThreading", []slackscot.AnswerOption{slackscot.AnswerWithoutThreading()}, map[string]string{slackscot.ThreadedReplyOpt: "false"}},
		{"threadReplyOnExistingThread", []slackscot.AnswerOption{slackscot.AnswerInExistingThread("1000")}, map[string]string{slackscot.ThreadedReplyOpt: "true", slackscot.ThreadTimestamp: "1000"}},
		{"ephemeralAnswer", []slackscot.AnswerOption{slackscot.AnswerEphemeral("U12321")}, map[string]string{slackscot.EphemeralAnswerToOpt: "U12321"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := slackscot.ApplyAnswerOpts(tc.options...)
			assert.Equal(t, tc.expectedConfig, c)
		})
	}
}
