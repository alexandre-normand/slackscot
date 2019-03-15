package assertplugin_test

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/schedule"
	"github.com/alexandre-normand/slackscot/test/assertanswer"
	"github.com/alexandre-normand/slackscot/test/assertplugin"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"testing"
)

type myLittleTester struct {
	slackscot.Plugin
}

func newLittleTester() (mlt *myLittleTester) {
	mlt = new(myLittleTester)
	mlt.Name = "myLittleTester"

	mlt.Commands = []slackscot.ActionDefinition{{
		Hidden: true,
		Match: func(m *slackscot.IncomingMessage) bool {
			return strings.HasPrefix(m.NormalizedText, "tell me where the black-capped chickadee is")
		},
		Usage:       "",
		Description: "",
		Answer:      mlt.findChicakee,
	}}

	mlt.HearActions = []slackscot.ActionDefinition{
		{
			Hidden: true,
			Match: func(m *slackscot.IncomingMessage) bool {
				return strings.Contains(m.NormalizedText, "are you up?")
			},
			Usage:       "",
			Description: "",
			Answer:      areYouAnswerer,
		},
		{
			Hidden: true,
			Match: func(m *slackscot.IncomingMessage) bool {
				return strings.Contains(m.NormalizedText, "hey")
			},
			Usage:       "",
			Description: "",
			Answer:      heyAnswerer,
		},
		{
			Hidden: true,
			Match: func(m *slackscot.IncomingMessage) bool {
				return strings.Contains(m.NormalizedText, "blue jays")
			},
			Usage:       "",
			Description: "",
			Answer:      mlt.emojiReact,
		},
	}

	mlt.ScheduledActions = []slackscot.ScheduledActionDefinition{
		{Schedule: schedule.Definition{Interval: 1, Unit: schedule.Minutes}, Description: "Check health", Action: mlt.healthStatus},
	}

	return mlt
}

func (mlt *myLittleTester) findChicakee(m *slackscot.IncomingMessage) *slackscot.Answer {
	mlt.Logger.Debugf("a debug statement")
	mlt.FileUploader.UploadFile(slack.FileUploadParameters{Filename: "imageOfABirdInATree.png", Filetype: "image/png", Title: "Look"})

	return &slackscot.Answer{Text: "👀 in the 🌲"}
}

func areYouAnswerer(m *slackscot.IncomingMessage) *slackscot.Answer {
	return &slackscot.Answer{Text: "I'm 😴, you?"}
}

func heyAnswerer(m *slackscot.IncomingMessage) *slackscot.Answer {
	return &slackscot.Answer{Text: "hey wut?"}
}

func (mlt *myLittleTester) healthStatus(sender slackscot.RealTimeMessageSender) {
	sender.SendNewMessage("test", "healthy")
	mlt.FileUploader.UploadFile(slack.FileUploadParameters{Filename: "healthStatus.png", Filetype: "image/png", Title: "healthy"})
}

func (mlt *myLittleTester) emojiReact(m *slackscot.IncomingMessage) *slackscot.Answer {
	mlt.EmojiReactor.AddReaction("owl", slack.NewRefToMessage(m.Channel, m.Timestamp))

	return nil
}

func TestCommandResultNonValid(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New(mockT, "bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, false, assertplugin.AnswersAndReacts(&myLittleTester.Plugin, &slack.Msg{Text: "<@bot> tell me where the black-capped chickadee is"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 10)
	}))
}

func TestCommandResultValid(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New(mockT, "bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.AnswersAndReacts(&myLittleTester.Plugin, &slack.Msg{Text: "<@bot> tell me where the black-capped chickadee is"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "👀 in the 🌲")
	}))
}

func TestFileUploadCapture(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New(mockT, "bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.AnswersAndReactsWithUploads(&myLittleTester.Plugin, &slack.Msg{Text: "<@bot> tell me where the black-capped chickadee is"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string, fileUploads []slack.FileUploadParameters) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "👀 in the 🌲") && assert.Len(t, fileUploads, 1) && assert.Equal(t, slack.FileUploadParameters{Filename: "imageOfABirdInATree.png", Filetype: "image/png", Title: "Look"}, fileUploads[0])
	}))
}

func TestLoggerAttached(t *testing.T) {
	mockT := new(testing.T)

	var b strings.Builder
	logger := log.New(&b, "", 0)
	assertplugin := assertplugin.New(mockT, "bot", assertplugin.OptionLog(logger))
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.AnswersAndReacts(&myLittleTester.Plugin, &slack.Msg{Text: "<@bot> tell me where the black-capped chickadee is"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "👀 in the 🌲") && assert.Equal(t, "a debug statement\n", b.String())
	}))
}

func TestHearResultValid(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New(mockT, "bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.AnswersAndReacts(&myLittleTester.Plugin, &slack.Msg{Text: "are you up?"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "I'm 😴, you?")
	}))
}

func TestDirectCommandResultValid(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New(mockT, "bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.AnswersAndReacts(&myLittleTester.Plugin, &slack.Msg{Text: "tell me where the black-capped chickadee is", Channel: "DTOTHEBOT"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "👀 in the 🌲")
	}))
}

func TestEmojiReaction(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New(mockT, "bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.AnswersAndReacts(&myLittleTester.Plugin, &slack.Msg{Text: "blue jays"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers) && assert.Contains(t, emojis, "owl")
	}))
}

func TestMultipleAnswersWithEmojiReaction(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New(mockT, "bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.AnswersAndReacts(&myLittleTester.Plugin, &slack.Msg{Text: "hey, are you up? I think I just saw blue jays"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 2) && assertanswer.HasText(t, answers[0], "I'm 😴, you?") && assertanswer.HasText(t, answers[1], "hey wut?") && assert.Contains(t, emojis, "owl")
	}))
}

func TestRunsOnScheduleAssert(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New(mockT, "bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.RunsOnSchedule(&myLittleTester.Plugin, schedule.Definition{Interval: 1, Unit: schedule.Minutes}, func(t *testing.T, sentMsgs map[string][]string, fileUploads []slack.FileUploadParameters) bool {
		return assert.Len(t, sentMsgs, 1) && assert.Contains(t, sentMsgs, "test") && assert.Contains(t, sentMsgs["test"], "healthy") && assert.Len(t, fileUploads, 1) && assert.Equal(t, slack.FileUploadParameters{Filename: "healthStatus.png", Filetype: "image/png", Title: "healthy"}, fileUploads[0])
	}))
}

func TestRunsOnScheduleAssertWhenDoesNotRun(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New(mockT, "bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, false, assertplugin.RunsOnSchedule(&myLittleTester.Plugin, schedule.Definition{Interval: 1, Unit: schedule.Hours}, func(t *testing.T, sentMsgs map[string][]string, fileUploads []slack.FileUploadParameters) bool {
		return true
	}))
}

func TestRunsOnScheduleAssertFailingValidator(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New(mockT, "bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, false, assertplugin.RunsOnSchedule(&myLittleTester.Plugin, schedule.Definition{Interval: 1, Unit: schedule.Minutes}, func(t *testing.T, sentMsgs map[string][]string, fileUploads []slack.FileUploadParameters) bool {
		return assert.Contains(t, sentMsgs, "myOtherChannel")
	}))
}

func TestDoesNotOnScheduleAssert(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New(mockT, "bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, true, assertplugin.DoesNotRunOnSchedule(&myLittleTester.Plugin, schedule.Definition{Interval: 1, Unit: schedule.Hours}))
}

func TestDoesNotOnScheduleAssertWhenRunsOnSchedule(t *testing.T) {
	mockT := new(testing.T)
	assertplugin := assertplugin.New(mockT, "bot")
	myLittleTester := newLittleTester()

	assert.Equal(t, false, assertplugin.DoesNotRunOnSchedule(&myLittleTester.Plugin, schedule.Definition{Interval: 1, Unit: schedule.Minutes}))
}
