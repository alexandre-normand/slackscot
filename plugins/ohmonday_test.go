package plugins_test

import (
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/schedule"
	"github.com/alexandre-normand/slackscot/test/assertplugin"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSendValidGreetingEachTimeCalled(t *testing.T) {
	pc := viper.New()
	pc.Set("channelIDs", []string{"channel1", "channel2"})
	pc.Set("atTime", "09:00")

	o, err := plugins.NewOhMonday(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")

	for i := 0; i < 100; i++ {
		assertplugin.RunsOnSchedule(&o.Plugin, schedule.Definition{Interval: 1, Weekday: time.Monday.String(), Unit: schedule.Weeks, AtTime: "09:00"}, func(t *testing.T, sentMsgs map[string][]string) bool {
			return assert.Contains(t, sentMsgs, "channel1") && assert.Len(t, sentMsgs["channel1"], 1) && assert.Contains(t, sentMsgs["channel1"][0], "https://") &&
				assert.Contains(t, sentMsgs, "channel2") && assert.Len(t, sentMsgs["channel2"], 1) && assert.Contains(t, sentMsgs["channel2"][0], "https://")
		})
	}
}

func TestDefaultAtTime(t *testing.T) {
	pc := viper.New()
	pc.Set("channelIDs", "testChannel")

	o, err := plugins.NewOhMonday(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")
	assertplugin.RunsOnSchedule(&o.Plugin, schedule.Definition{Interval: 1, Weekday: time.Monday.String(), Unit: schedule.Weeks, AtTime: "10:00"}, func(t *testing.T, sentMsgs map[string][]string) bool {
		return true
	})
}

func TestMissingChannelIDs(t *testing.T) {
	pc := viper.New()

	o, err := plugins.NewOhMonday(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")
	assertplugin.RunsOnSchedule(&o.Plugin, schedule.Definition{Interval: 1, Weekday: time.Monday.String(), Unit: schedule.Weeks, AtTime: "10:00"}, func(t *testing.T, sentMsgs map[string][]string) bool {
		return assert.Empty(t, sentMsgs)
	})
}

func TestEmptyChannels(t *testing.T) {
	pc := viper.New()
	pc.Set("channelIDs", "")

	o, err := plugins.NewOhMonday(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")
	assertplugin.RunsOnSchedule(&o.Plugin, schedule.Definition{Interval: 1, Weekday: time.Monday.String(), Unit: schedule.Weeks, AtTime: "10:00"}, func(t *testing.T, sentMsgs map[string][]string) bool {
		return assert.Empty(t, sentMsgs)
	})
}

func TestAtTimeOverride(t *testing.T) {
	pc := viper.New()
	pc.Set("channelIDs", "testChannel")
	pc.Set("atTime", "11:00")

	o, err := plugins.NewOhMonday(pc)
	assert.NoError(t, err)

	assertplugin := assertplugin.New(t, "bot")
	assertplugin.RunsOnSchedule(&o.Plugin, schedule.Definition{Interval: 1, Weekday: time.Monday.String(), Unit: schedule.Weeks, AtTime: "11:00"}, func(t *testing.T, sentMsgs map[string][]string) bool {
		return true
	})
}
