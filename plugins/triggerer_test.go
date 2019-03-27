package plugins_test

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/test/assertanswer"
	"github.com/alexandre-normand/slackscot/test/assertplugin"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"testing"
)

func TestRegisterNewTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	triggerer := plugins.NewTriggerer(mockStorer)

	mockStorer.On("GetSiloString", "myLittleChan", "Sdeal with it").Return("", fmt.Errorf("not found"))
	mockStorer.On("PutSiloString", "myLittleChan", "Sdeal with it", "http://dealwithit.gif").Return(nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{"Sdeal with it": "http://dealwithit.gif"}, nil)
	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)

	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> trigger on deal with it with http://dealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`deal with it` => `http://dealwithit.gif`]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with nothing"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, answers) && assert.Empty(t, emojis)
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with itself"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, answers) && assert.Empty(t, emojis)
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "http://dealwithit.gif") && assertanswer.HasOptions(t, answers[0])
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "DEAL WITH IT"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "http://dealwithit.gif") && assertanswer.HasOptions(t, answers[0])
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "don't tell me to deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "http://dealwithit.gif") && assertanswer.HasOptions(t, answers[0])
		})

	}
}

func TestTriggerReactionWithCollidingGlobalAndChannelTriggers(t *testing.T) {
	mockStorer := &mockStorer{}
	triggerer := plugins.NewTriggerer(mockStorer)

	mockStorer.On("ScanSilo", "").Return(map[string]string{"Sdeal with it": "http://global.gif"}, nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{"Sdeal with it": "http://channel.gif"}, nil)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "http://channel.gif") && assertanswer.HasOptions(t, answers[0])
	})
}

func TestTriggerReactionWithOnlyGlobalTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	triggerer := plugins.NewTriggerer(mockStorer)

	mockStorer.On("ScanSilo", "otherChan").Return(map[string]string{}, nil)
	mockStorer.On("ScanSilo", "").Return(map[string]string{"Sdeal with it": "http://global.gif"}, nil)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "otherChan", Text: "DEAL WITH IT"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "http://global.gif") && assertanswer.HasOptions(t, answers[0])
	})
}

func TestRegisterNewMultilineReactionTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	triggerer := plugins.NewTriggerer(mockStorer)

	mockStorer.On("GetSiloString", "myLittleChan", "Sdeal with it").Return("", fmt.Errorf("not found"))
	mockStorer.On("PutSiloString", "myLittleChan", "Sdeal with it", "```{\n\"attributes\"=1.0\n}\n```").Return(nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{"Sdeal with it": "```{\n\"attributes\"=1.0\n}\n```"}, nil)
	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)

	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> trigger on deal with it with ```{\n\"attributes\"=1.0\n}\n```"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`deal with it` => ```{\n\"attributes\"=1.0\n}\n```]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "```{\n\"attributes\"=1.0\n}\n```") && assertanswer.HasOptions(t, answers[0])
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "DEAL WITH IT"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "```{\n\"attributes\"=1.0\n}\n```") && assertanswer.HasOptions(t, answers[0])
		})
	}
}

func TestRegisterNewEmojiTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	triggerer := plugins.NewTriggerer(mockStorer)

	mockStorer.On("GetSiloString", "myLittleChan", "Edeal with it").Return("", fmt.Errorf("not found"))
	mockStorer.On("PutSiloString", "myLittleChan", "Edeal with it", "boom,cat").Return(nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{"Edeal with it": "boom,cat"}, nil)
	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)

	assertplugin := assertplugin.New(t, "bot")

	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> emoji trigger on deal with it with :boom::cat:"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new emoji trigger [`deal with it` => :boom:, :cat:]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with nothing"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, emojis) && assert.Empty(t, answers)
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with itself"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, emojis) && assert.Empty(t, answers)
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, answers) && assert.Contains(t, emojis, "boom", "cat")
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "DEAL WITH IT"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, answers) && assert.Contains(t, emojis, "boom", "cat")
		})
	}
}

func TestRegisterNewEmojiTriggerWithoutEmojis(t *testing.T) {
	mockStorer := &mockStorer{}
	triggerer := plugins.NewTriggerer(mockStorer)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> emoji trigger on deal with it with what are emojis?"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Invalid reaction for emoji trigger: `<reaction emojis>` doesn't include any emojis") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestErrorGettingTriggersWhenReacting(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)
	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{}, fmt.Errorf("error getting triggers"))

	triggerer := plugins.NewTriggerer(mockStorer)

	assertplugin := assertplugin.New(t, "bot")

	// Validate trigger reaction is absent because of the error listing triggers
	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers) && assert.Empty(t, emojis)
	})
}

func TestErrorGettingGlobalTriggersWhenReacting(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "").Return(map[string]string{}, fmt.Errorf("error getting triggers"))

	triggerer := plugins.NewTriggerer(mockStorer)

	assertplugin := assertplugin.New(t, "bot")

	// Validate trigger reaction is absent because of the error listing triggers
	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers) && assert.Empty(t, emojis)
	})
}

// This case interacts directly with the hear action since it's a little harder to an error listing triggers when Answering if we go through the usual
// MatchesAndAnswers flow. But since it's possible for no error to happen on Match and then an error to suddenly happen on Answer, we test that case here
func TestErrorGettingTriggersWhenAnswering(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)
	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{}, fmt.Errorf("error getting triggers"))

	triggerer := plugins.NewTriggerer(mockStorer)
	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Validate answer is empty because of the error listing triggers
	hearAction := triggerer.HearActions[0]
	assert.Nil(t, hearAction.Answer(&slackscot.IncomingMessage{NormalizedText: "deal with it", Msg: slack.Msg{Channel: "myLittleChan", Text: "deal with it"}}))
}

// This case interacts directly with the hear action since it's a little harder to simulate a case of Answer getting called without a matching trigger.
// But since it can still happen (a race condition and a trigger being deleted between Match and Answer being called), we want to test it
func TestNoReactionWhenNoTriggers(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)
	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{}, nil)

	triggerer := plugins.NewTriggerer(mockStorer)
	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	hearAction := triggerer.HearActions[0]
	assert.Nil(t, hearAction.Answer(&slackscot.IncomingMessage{NormalizedText: "deal with it", Msg: slack.Msg{Channel: "myLittleChan", Text: "deal with it"}}))
}

func TestNoAnswersAndNoEmojisWhenNoTriggers(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)
	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{}, nil)

	triggerer := plugins.NewTriggerer(mockStorer)
	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers) && assert.Empty(t, emojis)
	})
}

func TestErrorOnRegisterNewTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GetSiloString", "myLittleChan", "Sdeal with it").Return("", fmt.Errorf("not found"))
	mockStorer.On("PutSiloString", "myLittleChan", "Sdeal with it", "http://dealwithit.gif").Return(fmt.Errorf("Mock error"))

	triggerer := plugins.NewTriggerer(mockStorer)
	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger and get an error persisting it
	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> trigger on deal with it with http://dealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Error persisting standard trigger [`deal with it` => `http://dealwithit.gif`]: `Mock error`") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestUpdateTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GetSiloString", "myLittleChan", "Sdeal with it").Return("http://dealwithit.gif", nil)
	mockStorer.On("PutSiloString", "myLittleChan", "Sdeal with it", "http://betterdealwithit.gif").Return(nil)
	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{"Sdeal with it": "http://betterdealwithit.gif"}, nil)

	triggerer := plugins.NewTriggerer(mockStorer)
	assertplugin := assertplugin.New(t, "bot")

	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> trigger on deal with it with http://betterdealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Replaced standard trigger reaction for [`deal with it`] with [`http://betterdealwithit.gif`] (was [`http://dealwithit.gif`] previously)") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// Validate updated trigger reaction
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "http://betterdealwithit.gif") && assertanswer.HasOptions(t, answers[0])
		})
	}
}

func TestUpdateEmojiTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GetSiloString", "myLittleChan", "Edeal with it").Return("man-in-suit", nil)
	mockStorer.On("PutSiloString", "myLittleChan", "Edeal with it", "boom").Return(nil)
	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{"Edeal with it": "boom"}, nil)

	triggerer := plugins.NewTriggerer(mockStorer)
	assertplugin := assertplugin.New(t, "bot")

	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> emoji trigger on deal with it with :boom:"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Replaced emoji trigger reaction for [`deal with it`] with [:boom:] (was [:man-in-suit:] previously)") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// Validate updated trigger reaction
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, answers) && assert.Contains(t, emojis, "boom")
		})
	}
}

func TestErrorOnUpdateTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GetSiloString", "myLittleChan", "Sdeal with it").Return("http://dealwithit.gif", nil)
	mockStorer.On("PutSiloString", "myLittleChan", "Sdeal with it", "http://betterdealwithit.gif").Return(fmt.Errorf("Mock error"))

	triggerer := plugins.NewTriggerer(mockStorer)
	assertplugin := assertplugin.New(t, "bot")

	// Attempt to update trigger and expect error message
	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> trigger on deal with it with http://betterdealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Error persisting standard trigger [`deal with it` => `http://betterdealwithit.gif`]: `Mock error`") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestDeleteTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GetSiloString", "myLittleChan", "Sdeal with it").Return("http://dealwithit.gif", nil)
	mockStorer.On("DeleteSiloString", "myLittleChan", "Sdeal with it").Return(nil)

	triggerer := plugins.NewTriggerer(mockStorer)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> forget trigger on deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Deleted standard trigger [`deal with it` => `http://dealwithit.gif`]") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestDeleteGlobalTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	// No channel trigger
	mockStorer.On("GetSiloString", "myLittleChan", "Sdeal with it").Return("", fmt.Errorf("not found"))
	mockStorer.On("GetSiloString", "", "Sdeal with it").Return("http://global.gif", nil)
	mockStorer.On("DeleteSiloString", "", "Sdeal with it").Return(nil)

	triggerer := plugins.NewTriggerer(mockStorer)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> forget trigger on deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Deleted standard trigger [`deal with it` => `http://global.gif`]") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestDeleteEmojiTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GetSiloString", "myLittleChan", "Edeal with it").Return("boom", nil)
	mockStorer.On("DeleteSiloString", "myLittleChan", "Edeal with it").Return(nil)
	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{}, nil)

	triggerer := plugins.NewTriggerer(mockStorer)
	assertplugin := assertplugin.New(t, "bot")

	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> forget emoji trigger on deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Deleted emoji trigger [`deal with it` => :boom:]") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// Validate no reaction
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, emojis) && assert.Empty(t, answers)
		})
	}
}

func TestDeleteTriggerNotFound(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GetSiloString", "myLittleChan", "Sdeal with it").Return("", fmt.Errorf("not found"))
	mockStorer.On("GetSiloString", "", "Sdeal with it").Return("", fmt.Errorf("not found"))

	triggerer := plugins.NewTriggerer(mockStorer)

	assertplugin := assertplugin.New(t, "bot")

	// Delete trigger
	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> forget trigger on deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "No standard trigger found on `deal with it`") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestErrorOnDeleteTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GetSiloString", "myLittleChan", "Sdeal with it").Return("http://dealwithit.gif", nil)
	mockStorer.On("DeleteSiloString", "myLittleChan", "Sdeal with it").Return(fmt.Errorf("Mock error"))

	triggerer := plugins.NewTriggerer(mockStorer)
	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> forget trigger on deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Error removing standard trigger [`deal with it` => `http://dealwithit.gif`]: `Mock error`") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestErrorOnDeleteGlobalTrigger(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GetSiloString", "myLittleChan", "Sdeal with it").Return("", fmt.Errorf("not found"))
	mockStorer.On("GetSiloString", "", "Sdeal with it").Return("http://funnygif.gif", nil)
	mockStorer.On("DeleteSiloString", "", "Sdeal with it").Return(fmt.Errorf("Mock error"))

	triggerer := plugins.NewTriggerer(mockStorer)
	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> forget trigger on deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Error removing standard trigger [`deal with it` => `http://funnygif.gif`]: `Mock error`") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestListTriggers(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{"Sdeal with it": "http://dealwithit.gif", "Ssuddenly": "```{\n\"attributes\"=1.0\n}\n```"}, nil)

	triggerer := plugins.NewTriggerer(mockStorer)
	assertplugin := assertplugin.New(t, "bot")

	// List triggers
	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> list triggers"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Here are the current triggers: \n     • `deal with it` => `http://dealwithit.gif`\n     • `suddenly`     => ```{\n\"attributes\"=1.0\n}\n```\n\n") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestListEmojiTriggers(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{"Edeal with it": "sunglasses", "Esuddenly": "scream,ghost"}, nil)

	triggerer := plugins.NewTriggerer(mockStorer)
	assertplugin := assertplugin.New(t, "bot")

	// List triggers
	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> list emoji triggers"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Here are the current emoji triggers: \n     • `deal with it` => :sunglasses:\n     • `suddenly`     => :scream:, :ghost:\n\n") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestErrorOnListTriggers(t *testing.T) {
	mockStorer := &mockStorer{}
	defer mockStorer.AssertExpectations(t)
	mockStorer.On("ScanSilo", "").Return(map[string]string{}, nil)
	mockStorer.On("ScanSilo", "myLittleChan").Return(map[string]string{}, fmt.Errorf("Mock error"))

	triggerer := plugins.NewTriggerer(mockStorer)
	assertplugin := assertplugin.New(t, "bot")

	// List triggers
	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Channel: "myLittleChan", Text: "<@bot> list triggers"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Error loading triggers:\n```Mock error```") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}
