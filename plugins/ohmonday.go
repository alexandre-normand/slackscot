// Package plugins provides a collection of example (and usable) plugins for instances
// of slackscot
package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/v2"
	"github.com/alexandre-normand/slackscot/v2/config"
	"github.com/alexandre-normand/slackscot/v2/schedule"
	"github.com/alexandre-normand/slackscot/v2/slog"
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
	// OhMondayPluginName holds identifying name for the karma plugin
	OhMondayPluginName = "ohMonday"
)

const (
	defaultAtTime = "10:00"
)

var selectionRandom = rand.New(rand.NewSource(time.Now().Unix()))

// OhMonday holds the plugin data for the Oh Monday plugin
type OhMonday struct {
	slackscot.Plugin
	channelId string
}

// NewOhMonday creates a new instance of the OhMonday plugin
func NewOhMonday(c *config.PluginConfig) (o *OhMonday, err error) {
	c.SetDefault(atTimeKey, defaultAtTime)

	scheduleDefinition := schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Weeks, Weekday: time.Monday.String(), AtTime: c.GetString(atTimeKey)}

	if ok := c.IsSet(channelIdKey); !ok {
		return nil, fmt.Errorf("Missing [%s] configuration key for plugin [%s]", channelIdKey, OhMondayPluginName)
	}

	o = new(OhMonday)
	o.Name = OhMondayPluginName
	o.channelId = c.GetString(channelIdKey)
	o.ScheduledActions = []slackscot.ScheduledActionDefinition{{ScheduleDefinition: scheduleDefinition, Description: "Start the week off with a nice greeting", Action: o.sendGreeting}}

	return o, nil
}

func (o *OhMonday) sendGreeting(rtm *slack.RTM) {
	m := rtm.NewOutgoingMessage(mondayPictures[selectionRandom.Intn(len(mondayPictures))], o.channelId)
	slog.Debugf(o.Plugin.BotServices.Logger, "[%s] Sending morning greeting message [%s] to [%s]", OhMondayPluginName, m.Text, o.channelId)

	rtm.SendMessage(m)
}
