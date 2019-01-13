package plugins_test

import (
	"github.com/alexandre-normand/slackscot/v2/plugins"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSendValidVersionMessage(t *testing.T) {
	v := plugins.NewVersioner("little-red", "1.0.0")
	assert.NotNil(t, v)

	vc := v.Commands[0]

	msg := vc.Answer(&slack.Msg{})
	assert.Equal(t, "I'm `little-red`, version `1.0.0`", msg)
}

func TestMatchOnVersionCommand(t *testing.T) {
	v := plugins.NewVersioner("little-red", "1.0.0")
	assert.NotNil(t, v)

	vc := v.Commands[0]

	m := vc.Match("version", &slack.Msg{})
	assert.Equal(t, true, m)

	m = vc.Match(" version", &slack.Msg{})
	assert.Equal(t, false, m)

	m = vc.Match("version ", &slack.Msg{})
	assert.Equal(t, true, m)
}
