package test

import (
	"fmt"
	"github.com/nlopes/slack"
)

// EmojiReactionCaptor captures emoji reactions recorded by
// invocations of AddReaction. It only supports recording
// emojis for one given channel and timestamp
type EmojiReactionCaptor struct {
	Channel   string
	Timestamp string
	Emojis    []string
}

func (e *EmojiReactionCaptor) AddReaction(name string, item slack.ItemRef) error {
	if e.Channel == "" {
		e.Channel = item.Channel
		e.Timestamp = item.Timestamp
		e.Emojis = append(e.Emojis, name)
	} else if e.Channel == item.Channel && e.Timestamp == item.Timestamp {
		e.Emojis = append(e.Emojis, name)
	} else {
		return fmt.Errorf("EmojiReactionCaptor doesn't support capturing emojis for more than one message")
	}

	return nil
}

// NewEmojiReactionCaptor returns a new EmojiReactionCaptor with an initialized emojis array
func NewEmojiReactionCaptor() (emojiReactionCaptor *EmojiReactionCaptor) {
	emojiReactionCaptor = new(EmojiReactionCaptor)
	emojiReactionCaptor.Emojis = make([]string, 0)

	return emojiReactionCaptor
}
