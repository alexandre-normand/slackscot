package slackscot

import (
	"github.com/nlopes/slack"
)

// EmojiReactor is implemented by any value that has the AddReaction method.
// The main purpose is a slight decoupling of the slack.Client in order for plugins to
// be able to write cleaner tests more easily
type EmojiReactor interface {
	// AddReaction adds an emoji reaction to a ItemRef using the emoji associated
	// with the given name (i.e. name should be thumbsup rather than :thumbsup:)
	AddReaction(name string, item slack.ItemRef) error
}
