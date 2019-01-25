package plugins_test

import (
	"github.com/alexandre-normand/slackscot/v2"
	"github.com/alexandre-normand/slackscot/v2/plugins"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSendValidVersionMessage(t *testing.T) {
	v := plugins.NewVersioner("little-red", "1.0.0")
	assert.NotNil(t, v)

	vc := v.Commands[0]

	answer := vc.Answer(&slackscot.IncomingMessage{})
	assert.Equal(t, "I'm `little-red`, version `1.0.0`", answer.Text)
}

func TestMatchOnVersionCommand(t *testing.T) {
	v := plugins.NewVersioner("little-red", "1.0.0")
	assert.NotNil(t, v)

	vc := v.Commands[0]

	m := vc.Match(&slackscot.IncomingMessage{NormalizedText: "version"})
	assert.Equal(t, true, m)

	m = vc.Match(&slackscot.IncomingMessage{NormalizedText: " version"})
	assert.Equal(t, false, m)

	m = vc.Match(&slackscot.IncomingMessage{NormalizedText: "version "})
	assert.Equal(t, true, m)
}
