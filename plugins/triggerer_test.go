package plugins_test

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/test/assertaction"
	"github.com/alexandre-normand/slackscot/test/assertanswer"
	"github.com/nlopes/slack"
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

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Register new trigger
	registerCommand := triggerer.Commands[0]
	if assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on deal with it with http://dealwithit.gif", Msg: slack.Msg{Text: "@bot trigger on deal with it with http://dealwithit.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return assertanswer.HasText(t, a, "Registered new trigger `[deal with it => http://dealwithit.gif]`") &&
			assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// Validate trigger reaction
		hearAction := triggerer.HearActions[0]
		assertaction.NotMatch(t, hearAction, &slackscot.IncomingMessage{NormalizedText: "deal with nothing", Msg: slack.Msg{Text: "deal with nothing"}})
		assertaction.MatchesAndAnswers(t, hearAction, &slackscot.IncomingMessage{NormalizedText: "deal with it", Msg: slack.Msg{Text: "deal with it"}}, func(t *testing.T, a *slackscot.Answer) bool {
			return assertanswer.HasText(t, a, "http://dealwithit.gif") &&
				assertanswer.HasOptions(t, a)
		})

		assertaction.MatchesAndAnswers(t, hearAction, &slackscot.IncomingMessage{NormalizedText: "don't tell me to deal with it, I've already had enough", Msg: slack.Msg{Text: "don't tell me to deal with it, I've already had enough"}}, func(t *testing.T, a *slackscot.Answer) bool {
			return assertanswer.HasText(t, a, "http://dealwithit.gif") &&
				assertanswer.HasOptions(t, a)
		})
	}
}

func TestErrorGettingTriggersWhenReacting(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Register new trigger
	registerCommand := triggerer.Commands[0]
	if assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on deal with it with http://dealwithit.gif", Msg: slack.Msg{Text: "@bot trigger on deal with it with http://dealwithit.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return assertanswer.HasText(t, a, "Registered new trigger `[deal with it => http://dealwithit.gif]`") &&
			assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		storer.returnErrorOnReads = true

		// Validate trigger reaction is absent because of the error listing triggers
		hearAction := triggerer.HearActions[0]
		assertaction.NotMatch(t, hearAction, &slackscot.IncomingMessage{NormalizedText: "deal with it", Msg: slack.Msg{Text: "deal with it"}})
	}
}

// This case interacts directly with the hear action since it's a little harder to an error listing triggers when Answering if we go through the usual
// MatchesAndAnswers flow. But since it's possible for no error to happen on Match and then an error to suddenly happen on Answer, we test that case here
func TestErrorGettingTriggersWhenAnswering(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Register new trigger
	registerCommand := triggerer.Commands[0]
	if assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on deal with it with http://dealwithit.gif", Msg: slack.Msg{Text: "@bot trigger on deal with it with http://dealwithit.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return assertanswer.HasText(t, a, "Registered new trigger `[deal with it => http://dealwithit.gif]`") &&
			assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		storer.returnErrorOnReads = true

		// Validate answer is empty because of the error listing triggers
		hearAction := triggerer.HearActions[0]
		assertanswer.HasText(t, hearAction.Answer(&slackscot.IncomingMessage{NormalizedText: "deal with it", Msg: slack.Msg{Text: "deal with it"}}), "")
	}
}

// This case interacts directly with the hear action since it's a little harder to simulate a case of Answer getting called without a matching trigger.
// But since it can still happen (a race condition and a trigger being deleted between Match and Answer being called), we want to test it
func TestNoMoreTriggersWhenAnswering(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Validate answer is empty because of the error listing triggers
	hearAction := triggerer.HearActions[0]
	assertanswer.HasText(t, hearAction.Answer(&slackscot.IncomingMessage{NormalizedText: "deal with it", Msg: slack.Msg{Text: "deal with it"}}), "")
}

func TestErrorOnRegisterNewTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Have the storer return an error on write
	storer.returnErrorOnWrites = true

	// Register new trigger
	registerCommand := triggerer.Commands[0]
	assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on deal with it with http://dealwithit.gif", Msg: slack.Msg{Text: "@bot trigger on deal with it with http://dealwithit.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return assertanswer.HasText(t, a, "Error persisting trigger `[deal with it => http://dealwithit.gif]`: `Mock error`") &&
			assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

func TestUpdateTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Register new trigger
	registerCommand := triggerer.Commands[0]
	if assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on deal with it with http://dealwithit.gif", Msg: slack.Msg{Text: "@bot trigger on deal with it with http://dealwithit.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return assertanswer.HasText(t, a, "Registered new trigger `[deal with it => http://dealwithit.gif]`") &&
			assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// Update trigger
		if assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on deal with it with http://betterdealwithit.gif", Msg: slack.Msg{Text: "@bot trigger on deal with it with http://betterdealwithit.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
			return assertanswer.HasText(t, a, "Replaced trigger reaction for [`deal with it`] with [`http://betterdealwithit.gif`] (was [`http://dealwithit.gif`] previously)") &&
				assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		}) {
			// Validate trigger reaction
			hearAction := triggerer.HearActions[0]
			assertaction.MatchesAndAnswers(t, hearAction, &slackscot.IncomingMessage{NormalizedText: "deal with it", Msg: slack.Msg{Text: "deal with it"}}, func(t *testing.T, a *slackscot.Answer) bool {
				return assertanswer.HasText(t, a, "http://betterdealwithit.gif") &&
					assertanswer.HasOptions(t, a)
			})
		}
	}
}

func TestErrorOnUpdateTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Register new trigger
	registerCommand := triggerer.Commands[0]
	if assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on deal with it with http://dealwithit.gif", Msg: slack.Msg{Text: "@bot trigger on deal with it with http://dealwithit.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return assertanswer.HasText(t, a, "Registered new trigger `[deal with it => http://dealwithit.gif]`") &&
			assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// Have the storer return an error on write
		storer.returnErrorOnWrites = true

		// Attempt to update trigger
		assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on deal with it with http://dealwithit.gif", Msg: slack.Msg{Text: "@bot trigger on deal with it with http://dealwithit.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
			return assertanswer.HasText(t, a, "Error persisting trigger `[deal with it => http://dealwithit.gif]`: `Mock error`") &&
				assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		})
	}
}

func TestDeleteTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Register new trigger
	registerCommand := triggerer.Commands[0]
	if assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on deal with it with http://dealwithit.gif", Msg: slack.Msg{Text: "@bot trigger on deal with it with http://dealwithit.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return assertanswer.HasText(t, a, "Registered new trigger `[deal with it => http://dealwithit.gif]`") &&
			assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// Delete trigger
		deleteCommand := triggerer.Commands[1]
		if assertaction.MatchesAndAnswers(t, deleteCommand, &slackscot.IncomingMessage{NormalizedText: "forget trigger on deal with it", Msg: slack.Msg{Text: "@bot forget trigger on deal with it"}}, func(t *testing.T, a *slackscot.Answer) bool {
			return assertanswer.HasText(t, a, "Deleted trigger [`deal with it => http://dealwithit.gif`]") &&
				assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		}) {
			// Validate no reaction
			hearAction := triggerer.HearActions[0]
			assertaction.NotMatch(t, hearAction, &slackscot.IncomingMessage{NormalizedText: "deal with it", Msg: slack.Msg{Text: "deal with it"}})
		}
	}
}

func TestDeleteTriggerNotFound(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Delete trigger
	deleteCommand := triggerer.Commands[1]
	assertaction.MatchesAndAnswers(t, deleteCommand, &slackscot.IncomingMessage{NormalizedText: "forget trigger on deal with it", Msg: slack.Msg{Text: "@bot forget trigger on deal with it"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return assertanswer.HasText(t, a, "No trigger found on `deal with it`") &&
			assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	})
}

// TestAmbiguousDeleteTriggerNotTriggerNewOne validates a scenario where the delete could cause a new trigger to be registered "forget trigger on “Yeah. I agree with that also”"
func TestAmbiguousDeleteTriggerNotTriggerNewOne(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Check that the register command doesn't trigger on a delete trigger message
	registerCommand := triggerer.Commands[0]
	assertaction.NotMatch(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "forget trigger on “Yeah. I agree with that also”", Msg: slack.Msg{Text: "@bot forget trigger on “Yeah. I agree with that also”"}})
}

func TestErrorOnDeleteTrigger(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Register new trigger
	registerCommand := triggerer.Commands[0]
	if assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on deal with it with http://dealwithit.gif", Msg: slack.Msg{Text: "@bot trigger on deal with it with http://dealwithit.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return assertanswer.HasText(t, a, "Registered new trigger `[deal with it => http://dealwithit.gif]`") &&
			assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// Have the storer return an error on writes
		storer.returnErrorOnWrites = true

		// Delete trigger
		deleteCommand := triggerer.Commands[1]
		assertaction.MatchesAndAnswers(t, deleteCommand, &slackscot.IncomingMessage{NormalizedText: "forget trigger on deal with it", Msg: slack.Msg{Text: "@bot forget trigger on deal with it"}}, func(t *testing.T, a *slackscot.Answer) bool {
			return assertanswer.HasText(t, a, "Error removing trigger `[deal with it => http://dealwithit.gif]`: `Mock error`") &&
				assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		})
	}
}

func TestListTriggers(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Register triggers
	registerCommand := triggerer.Commands[0]
	if assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on deal with it with http://dealwithit.gif", Msg: slack.Msg{Text: "@bot trigger on deal with it with http://dealwithit.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return assertanswer.HasText(t, a, "Registered new trigger `[deal with it => http://dealwithit.gif]`") &&
			assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) && assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on suddenly with https://suddenly.gif", Msg: slack.Msg{Text: "@bot trigger on suddenly with https://suddenly.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return assertanswer.HasText(t, a, "Registered new trigger `[suddenly => https://suddenly.gif]`") &&
			assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// List triggers
		listCommand := triggerer.Commands[2]
		assertaction.MatchesAndAnswers(t, listCommand, &slackscot.IncomingMessage{NormalizedText: "list triggers", Msg: slack.Msg{Text: "@bot list triggers"}}, func(t *testing.T, a *slackscot.Answer) bool {
			return assertanswer.HasText(t, a, "Here are the current triggers: \n```deal with it => http://dealwithit.gif\nsuddenly     => https://suddenly.gif\n```\n") &&
				assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		})
	}
}

func TestErrorOnListTriggers(t *testing.T) {
	storer := newInMemoryStorer()
	triggerer := plugins.NewTriggerer(storer)

	var b strings.Builder
	triggerer.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	// Register new trigger
	registerCommand := triggerer.Commands[0]
	if assertaction.MatchesAndAnswers(t, registerCommand, &slackscot.IncomingMessage{NormalizedText: "trigger on deal with it with http://dealwithit.gif", Msg: slack.Msg{Text: "@bot trigger on deal with it with http://dealwithit.gif"}}, func(t *testing.T, a *slackscot.Answer) bool {
		return assertanswer.HasText(t, a, "Registered new trigger `[deal with it => http://dealwithit.gif]`") &&
			assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
	}) {
		// Have the storer return an error on reads
		storer.returnErrorOnReads = true

		// List triggers
		listCommand := triggerer.Commands[2]
		assertaction.MatchesAndAnswers(t, listCommand, &slackscot.IncomingMessage{NormalizedText: "list triggers", Msg: slack.Msg{Text: "@bot list triggers"}}, func(t *testing.T, a *slackscot.Answer) bool {
			return assertanswer.HasText(t, a, "Error loading triggers:\n```Mock error```") &&
				assertanswer.HasOptions(t, a, assertanswer.ResolvedAnswerOption{Key: slackscot.ThreadedReplyOpt, Value: "true"}, assertanswer.ResolvedAnswerOption{Key: slackscot.BroadcastOpt, Value: "false"})
		})
	}
}
