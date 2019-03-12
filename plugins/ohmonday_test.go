package plugins_test

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/schedule"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"testing"
	"time"
)

type FakeSender struct {
	msgs map[string]string
}

func (f *FakeSender) SendNewMessage(message string, channelID string) (err error) {
	f.msgs[channelID] = message

	return nil
}

func (f *FakeSender) GetAPI() (rtm *slack.RTM) {
	return nil
}

func TestSendValidGreetingEachTimeCalled(t *testing.T) {
	pc := viper.New()
	pc.Set("channelIDs", []string{"channel1", "channel2"})

	o, err := plugins.NewOhMonday(pc)
	assert.NoError(t, err)

	sa := o.ScheduledActions[0]
	var b strings.Builder
	o.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

	sender := FakeSender{msgs: make(map[string]string)}

	for i := 0; i < 100; i++ {
		sa.Action(&sender)

		if assert.Contains(t, sender.msgs, "channel1") {
			assert.Contains(t, sender.msgs["channel1"], "https://")
		}

		if assert.Contains(t, sender.msgs, "channel2") {
			assert.Contains(t, sender.msgs["channel2"], "https://")
		}
	}
}

func TestDefaultAtTime(t *testing.T) {
	pc := viper.New()
	pc.Set("channelIDs", "testChannel")

	o, err := plugins.NewOhMonday(pc)
	assert.NoError(t, err)
	sa := o.ScheduledActions[0]

	assert.Equal(t, schedule.Definition{Interval: 1, Weekday: time.Monday.String(), Unit: schedule.Weeks, AtTime: "10:00"}, sa.Schedule)
}

func TestMissingChannelIDs(t *testing.T) {
	pc := viper.New()

	o, err := plugins.NewOhMonday(pc)
	assert.NoError(t, err)

	sa := o.ScheduledActions[0]
	var b strings.Builder
	o.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)
	sender := FakeSender{msgs: make(map[string]string)}

	sa.Action(&sender)

	assert.Empty(t, sender.msgs)
}

func TestEmptyChannels(t *testing.T) {
	pc := viper.New()
	pc.Set("channelIDs", "")

	o, err := plugins.NewOhMonday(pc)
	assert.NoError(t, err)

	sa := o.ScheduledActions[0]
	var b strings.Builder
	o.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)
	sender := FakeSender{msgs: make(map[string]string)}

	sa.Action(&sender)

	assert.Empty(t, sender.msgs)
}

func TestAtTimeOverride(t *testing.T) {
	pc := viper.New()
	pc.Set("channelIDs", "testChannel")
	pc.Set("atTime", "11:00")

	o, err := plugins.NewOhMonday(pc)
	assert.NoError(t, err)
	sa := o.ScheduledActions[0]

	assert.Equal(t, schedule.Definition{Interval: 1, Weekday: time.Monday.String(), Unit: schedule.Weeks, AtTime: "11:00"}, sa.Schedule)

}
