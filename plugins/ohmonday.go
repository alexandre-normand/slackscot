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
	defaultAtTime      = "10:00"
	OhMondayPluginName = "ohMonday"
)

// OhMonday holds the plugin data for the Oh Monday plugin
type OhMonday struct {
	slackscot.Plugin
}

// NewOhMonday creates a new instance of the OhMonday plugin
func NewOhMonday(c *config.PluginConfig) (p *OhMonday, err error) {
	c.SetDefault(atTimeKey, defaultAtTime)

	scheduleDefinition := schedule.ScheduleDefinition{Interval: 1, Unit: schedule.WEEKS, Weekday: time.Monday.String(), AtTime: c.GetString(atTimeKey)}

	if ok := c.IsSet(channelIdKey); !ok {
		return nil, fmt.Errorf("Missing [%s] configuration key for plugin [%s]", channelIdKey, OhMondayPluginName)
	}

	channelId := c.GetString(channelIdKey)

	selectionRandom := rand.New(rand.NewSource(time.Now().Unix()))

	mondayGreeting := slackscot.ScheduledActionDefinition{
		ScheduleDefinition: scheduleDefinition, Description: "Start the week off with a nice greeting", Action: func(rtm *slack.RTM) {
			o := rtm.NewOutgoingMessage(mondayPictures[selectionRandom.Intn(len(mondayPictures))], channelId)
			slackscot.Debugf("[%s] About to send message [%s] to [%s]", o.Text, channelId)

			rtm.SendMessage(o)
		}}

	plugin := OhMonday{Plugin: slackscot.Plugin{Name: OhMondayPluginName, Commands: nil, HearActions: nil, ScheduledActions: []slackscot.ScheduledActionDefinition{mondayGreeting}}}
	return &plugin, nil
}
