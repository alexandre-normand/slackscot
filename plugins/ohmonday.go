// Package plugins provides a collection of example (and usable) plugins for instances
// of slackscot
package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/schedule"
	"github.com/nlopes/slack"
	"math/rand"
	"time"
)

// Configuration keys
const (
	atTimeKey    = "atTime"
	channelIdKey = "channelId"
)

var defaultScheduleDefinition = schedule.ScheduleDefinition{Interval: 1, Unit: schedule.WEEKS, Weekday: time.Monday.String(), AtTime: defaultAtTime}
var mondayPictures = []string{"https://media.giphy.com/media/3og0IHx11gZBccA98c/giphy-downsized.gif",
	"https://media.giphy.com/media/vguRpQzGag7M5h4UVt/giphy-downsized.gif",
	"https://media.giphy.com/media/9GI7UlOQ6uU95v82q7/giphy-downsized.gif",
	"https://media.giphy.com/media/hu3Z1fwuOZh3a/giphy-downsized.gif",
	"https://media.giphy.com/media/5ZZSYqvcH6QppFQGI5/giphy-downsized.gif",
	"https://media.giphy.com/media/7mMRX7gWzDVwA/giphy-downsized.gif",
	"https://media.giphy.com/media/Mv6t9sASpgTEA/giphy.gif",
	"https://media.giphy.com/media/GGFMa2baxgoLK/giphy.gif",
	"https://media.giphy.com/media/WET6Ed65VUkuY/giphy-downsized.gif",
	"https://media.giphy.com/media/26wkRxKJ9yUZzlorK/giphy-downsized.gif",
}

const (
	ohMondayPluginName = "ohMonday"
	defaultAtTime      = "10:00"
)

// OhMonday holds the plugin data for the Oh Monday plugin
type OhMonday struct {
	slackscot.Plugin
}

// NewOhMonday creates a new instance of the OhMonday plugin
func NewOhMonday(config config.Configuration) (p *OhMonday, err error) {
	scheduleDefinition := defaultScheduleDefinition
	channel := ""

	if pluginConfig, ok := config.Plugins[ohMondayPluginName]; ok {
		if atTime, ok := pluginConfig[atTimeKey]; ok {
			scheduleDefinition.AtTime = atTime
		} else {
			slackscot.Debugf(config, "Missing [%s] configuration, will use default atTime of [%s]\n", atTimeKey, defaultAtTime)
		}

		if channelId, ok := pluginConfig[channelIdKey]; ok {
			channel = channelId
		} else {
			return nil, fmt.Errorf("Missing [%s] configuration key for plugin [%s]", channelIdKey, ohMondayPluginName)
		}
	} else {
		return nil, fmt.Errorf("Missing configuration for plugin [%s]", ohMondayPluginName)
	}

	selectionRandom := rand.New(rand.NewSource(time.Now().Unix()))

	mondayGreeting := slackscot.ScheduledActionDefinition{
		ScheduleDefinition: scheduleDefinition, Description: "Start the week of with a nice greeting", Action: func(rtm *slack.RTM) {
			o := rtm.NewOutgoingMessage(mondayPictures[selectionRandom.Intn(len(mondayPictures))], channel)
			slackscot.Debugf(config, "[%s] About to send message [%s] to [%s]", o.Text, channel)

			rtm.SendMessage(o)
		}}

	plugin := OhMonday{Plugin: slackscot.Plugin{Name: ohMondayPluginName, Commands: nil, HearActions: nil, ScheduledActions: []slackscot.ScheduledActionDefinition{mondayGreeting}}}
	return &plugin, nil
}
