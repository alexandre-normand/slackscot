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

type memoryStringStorer struct {
	triggers            map[string]string
	returnErrorOnReads  bool
	returnErrorOnWrites bool
}

func (m *memoryStringStorer) GetString(key string) (value string, err error) {
	if m.returnErrorOnReads {
		return "", fmt.Errorf("Mock error")
	}

	return m.triggers[key], nil
}

func (m *memoryStringStorer) PutString(key string, value string) (err error) {
	if m.returnErrorOnWrites {
		return fmt.Errorf("Mock error")
	}

	m.triggers[key] = value
	return nil
}

func (m *memoryStringStorer) DeleteString(key string) (err error) {
	if m.returnErrorOnWrites {
		return fmt.Errorf("Mock error")
	}

	delete(m.triggers, key)
	return nil
}

func (m *memoryStringStorer) Scan() (entries map[string]string, err error) {
	if m.returnErrorOnReads {
		return make(map[string]string), fmt.Errorf("Mock error")
	}

	return m.triggers, nil
}

func (m *memoryStringStorer) Close() (err error) {
	return nil
}

func newInMemoryStorer() (m *memoryStringStorer) {
	return &memoryStringStorer{triggers: make(map[string]string)}
}

func TestRegisterNewTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with http://dealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`deal with it` => `http://dealwithit.gif`]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with nothing"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, answers) && assert.Empty(t, emojis)
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with itself"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, answers) && assert.Empty(t, emojis)
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "http://dealwithit.gif") && assertanswer.HasOptions(t, answers[0])
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "DEAL WITH IT"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "http://dealwithit.gif") && assertanswer.HasOptions(t, answers[0])
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "don't tell me to deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "http://dealwithit.gif") && assertanswer.HasOptions(t, answers[0])
		})

	}
}

func TestRegisterNewMultilineReactionTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with ```{\n\"attributes\"=1.0\n}\n```"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`deal with it` => ```{\n\"attributes\"=1.0\n}\n```]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "```{\n\"attributes\"=1.0\n}\n```") && assertanswer.HasOptions(t, answers[0])
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "DEAL WITH IT"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "```{\n\"attributes\"=1.0\n}\n```") && assertanswer.HasOptions(t, answers[0])
		})
	}
}

func TestRegisterNewEmojiTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	assertplugin := assertplugin.New(t, "bot")

	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> emoji trigger on deal with it with :boom::cat:"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new emoji trigger [`deal with it` => :boom:, :cat:]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with nothing"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, emojis) && assert.Empty(t, answers)
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with itself"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, emojis) && assert.Empty(t, answers)
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, answers) && assert.Contains(t, emojis, "boom", "cat")
		})

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "DEAL WITH IT"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, answers) && assert.Contains(t, emojis, "boom", "cat")
		})
	}
}

func TestRegisterNewEmojiTriggerWithoutEmojis(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> emoji trigger on deal with it with what are emojis?"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Invalid reaction for emoji trigger: `<reaction emojis>` doesn't include any emojis") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestErrorGettingTriggersWhenReacting(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with http://dealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`deal with it` => `http://dealwithit.gif`]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		storer.returnErrorOnReads = true

		// Validate trigger reaction is absent because of the error listing triggers
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, answers) && assert.Empty(t, emojis)
		})
	}
}

// This case interacts directly with the hear action since it's a little harder to an error listing triggers when Answering if we go through the usual
// MatchesAndAnswers flow. But since it's possible for no error to happen on Match and then an error to suddenly happen on Answer, we test that case here
func TestErrorGettingTriggersWhenAnswering(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)
	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with http://dealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`deal with it` => `http://dealwithit.gif`]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		storer.returnErrorOnReads = true

		// Validate answer is empty because of the error listing triggers
		hearAction := triggerer.HearActions[0]
		assert.Nil(t, hearAction.Answer(&slackscot.IncomingMessage{NormalizedText: "deal with it", Msg: slack.Msg{Text: "deal with it"}}))
	}
}

// This case interacts directly with the hear action since it's a little harder to simulate a case of Answer getting called without a matching trigger.
// But since it can still happen (a race condition and a trigger being deleted between Match and Answer being called), we want to test it
func TestNoReactionWhenNoTriggers(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	hearAction := triggerer.HearActions[0]
	assert.Nil(t, hearAction.Answer(&slackscot.IncomingMessage{NormalizedText: "deal with it", Msg: slack.Msg{Text: "deal with it"}}))
}

func TestNoAnswersAndNoEmojisWhenNoTriggers(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)
	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers) && assert.Empty(t, emojis)
	})
}

func TestErrorOnRegisterNewTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)
	assertplugin := assertplugin.New(t, "bot")

	// Have the storer return an error on write
	storer.returnErrorOnWrites = true

	// Register new trigger and get an error persisting it
	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with http://dealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Error persisting standard trigger [`deal with it` => `http://dealwithit.gif`]: `Mock error`") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestUpdateTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)
	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with http://dealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`deal with it` => `http://dealwithit.gif`]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with http://betterdealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Replaced standard trigger reaction for [`deal with it`] with [`http://betterdealwithit.gif`] (was [`http://dealwithit.gif`] previously)") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		}) {
			// Validate updated trigger reaction
			assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
				return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "http://betterdealwithit.gif") && assertanswer.HasOptions(t, answers[0])
			})
		}
	}
}

func TestUpdateEmojiTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)
	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> emoji trigger on deal with it with :man-in-suit:"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new emoji trigger [`deal with it` => :man-in-suit:]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> emoji trigger on deal with it with :boom:"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Replaced emoji trigger reaction for [`deal with it`] with [:boom:] (was [:man-in-suit:] previously)") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		}) {
			// Validate updated trigger reaction
			assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
				return assert.Empty(t, answers) && assert.Contains(t, emojis, "boom")
			})
		}
	}
}

func TestErrorOnUpdateTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)
	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with http://dealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`deal with it` => `http://dealwithit.gif`]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// Have the storer return an error on write
		storer.returnErrorOnWrites = true

		// Attempt to update trigger and expect error message
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with http://betterdealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Error persisting standard trigger [`deal with it` => `http://betterdealwithit.gif`]: `Mock error`") &&
				assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		})
	}
}

func TestDeleteTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with http://dealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`deal with it` => `http://dealwithit.gif`]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> forget trigger on deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Deleted standard trigger [`deal with it` => `http://dealwithit.gif`]") &&
				assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		}) {
			// Validate no reaction
			assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
				return assert.Empty(t, emojis) && assert.Empty(t, answers)
			})
		}
	}
}

func TestDeleteEmojiTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)
	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> emoji trigger on deal with it with :boom:"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new emoji trigger [`deal with it` => :boom:]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> forget emoji trigger on deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Deleted emoji trigger [`deal with it` => :boom:]") &&
				assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		}) {
			// Validate no reaction
			assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
				return assert.Empty(t, emojis) && assert.Empty(t, answers)
			})
		}
	}
}

func TestDeleteTriggerNotFound(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	assertplugin := assertplugin.New(t, "bot")

	// Delete trigger
	assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> forget trigger on deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "No standard trigger found on `deal with it`") &&
			assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestErrorOnDeleteTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)
	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with http://dealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`deal with it` => `http://dealwithit.gif`]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// Have the storer return an error on writes
		storer.returnErrorOnWrites = true

		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> forget trigger on deal with it"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Error removing standard trigger [`deal with it` => `http://dealwithit.gif`]: `Mock error`") &&
				assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		})
	}
}

func TestListTriggers(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)
	assertplugin := assertplugin.New(t, "bot")

	// Register triggers
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with http://dealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`deal with it` => `http://dealwithit.gif`]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) && assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on suddenly with ```{\n\"attributes\"=1.0\n}\n```"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`suddenly` => ```{\n\"attributes\"=1.0\n}\n```]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// List triggers
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> list triggers"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Here are the current triggers: \n     • `deal with it` => `http://dealwithit.gif`\n     • `suddenly`     => ```{\n\"attributes\"=1.0\n}\n```\n\n") &&
				assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		})
	}
}

func TestListEmojiTriggers(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)
	assertplugin := assertplugin.New(t, "bot")

	// Register triggers
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> emoji trigger on deal with it with :sunglasses:"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new emoji trigger [`deal with it` => :sunglasses:]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) && assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> emoji trigger on suddenly with :scream::ghost:"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new emoji trigger [`suddenly` => :scream:, :ghost:]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// List triggers
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> list emoji triggers"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Here are the current emoji triggers: \n     • `deal with it` => :sunglasses:\n     • `suddenly`     => :scream:, :ghost:\n\n") &&
				assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		})
	}
}

func TestErrorOnListTriggers(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)
	assertplugin := assertplugin.New(t, "bot")

	// Register new trigger
	if assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> trigger on deal with it with http://dealwithit.gif"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Registered new standard trigger [`deal with it` => `http://dealwithit.gif`]") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// Have the storer return an error on reads
		storer.returnErrorOnReads = true

		// List triggers
		assertplugin.AnswersAndReacts(&triggerer.Plugin, &slack.Msg{Text: "<@bot> list triggers"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
			return assert.Empty(t, emojis) && assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Error loading triggers:\n```Mock error```") &&
				assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		})
	}
}
