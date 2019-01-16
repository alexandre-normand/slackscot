package plugins_test

import (
	"github.com/alexandre-normand/slackscot/v2"
	"github.com/alexandre-normand/slackscot/v2/plugins"
	"github.com/alexandre-normand/slackscot/v2/schedule"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"testing"
	"time"
)

type FakeSender struct {
	message string
	channel string
}

func (f *FakeSender) SendNewMessage(message string, channelId string) (err error) {
	f.message = message
	f.channel = channelId

	return nil
}

func (f *FakeSender) GetAPI() (rtm *slack.RTM) {
	return nil
}

func TestSendValidGreetingEachTimeCalled(t *testing.T) {
	pc := viper.New()
	pc.Set("channelId", "testChannel")

	o, err := plugins.NewOhMonday(pc)
	assert.Nil(t, err)

	sa := o.ScheduledActions[0]
	var b strings.Builder
	o.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	sender := FakeSender{}

	for i := 0; i < 100; i++ {
		sa.Action(&sender)

		assert.Contains(t, sender.message, "https://")
		assert.Contains(t, sender.channel, "testChannel")
	}
}

func TestDefaultAtTime(t *testing.T) {
	pc := viper.New()
	pc.Set("channelId", "testChannel")

	o, err := plugins.NewOhMonday(pc)
	assert.Nil(t, err)
	sa := o.ScheduledActions[0]

	assert.Equal(t, schedule.Definition{Interval: 1, Weekday: time.Monday.String(), Unit: schedule.Weeks, AtTime: "10:00"}, sa.Schedule)
}

func TestMissingChannelId(t *testing.T) {
	pc := viper.New()

	_, err := plugins.NewOhMonday(pc)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "Missing [channelId] configuration key")
	}
}

func TestAtTimeOverride(t *testing.T) {
	pc := viper.New()
	pc.Set("channelId", "testChannel")
	pc.Set("atTime", "11:00")

	o, err := plugins.NewOhMonday(pc)
	assert.Nil(t, err)
	sa := o.ScheduledActions[0]

	assert.Equal(t, schedule.Definition{Interval: 1, Weekday: time.Monday.String(), Unit: schedule.Weeks, AtTime: "11:00"}, sa.Schedule)

}
