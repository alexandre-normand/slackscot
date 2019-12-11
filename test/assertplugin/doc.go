// Package assertplugin provides testing functions to validate a plugin's overall functionality.
// This package is designed to play well but not require the assertanswer package for validation
// of answers
//
// Note that all commands and hearActions are evaluated by assertplugin's driver but this is a
// simplified version of how slackscot actually drives plugins and aims to provide the minimal
// processing required to allow a plugin to test functionality given an incoming message.
// Users should take special care to use include <@botUserID> with the same botUserID with which the
// plugin driver has been instantiated in the message text inputs to test commands (or include a
// channel name that starts with D for direct channel testing)
//
// Example:
//    func TestPlugin(t *testing.T) {
//        assertplugin := assertplugin.New(t, "bot")
//        yourPlugin := newPlugin()
//
//        assertplugin.AnswersAndReacts(yourPlugin, &slack.Msg{Text: "are you up?"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
// 	          return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "I'm 😴, you?")
//        }))
//    }
package assertplugin // import "github.com/alexandre-normand/slackscot/test/assertplugin"
