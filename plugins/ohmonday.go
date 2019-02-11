// Package plugins provides a collection of example (and usable) plugins for instances
// of slackscot
package plugins

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/schedule"
	"math/rand"
	"time"
)

// Configuration keys
const (
	atTimeKey             = "atTime"
	ohMondayChannelIDsKey = "channelIDs"
)

var mondayPictures = []string{"https://media.giphy.com/media/3og0IHx11gZBccA98c/giphy-downsized-large.gif",
	"https://media.giphy.com/media/vguRpQzGag7M5h4UVt/giphy.gif",
	"https://media.giphy.com/media/9GI7UlOQ6uU95v82q7/giphy-downsized-large.gif",
	"https://media.giphy.com/media/hu3Z1fwuOZh3a/giphy.gif",
	"https://media.giphy.com/media/5ZZSYqvcH6QppFQGI5/giphy-downsized-large.gif",
	"https://media.giphy.com/media/7mMRX7gWzDVwA/giphy.gif",
	"https://media.giphy.com/media/Mv6t9sASpgTEA/giphy.gif",
	"https://media.giphy.com/media/GGFMa2baxgoLK/giphy.gif",
	"https://media.giphy.com/media/WET6Ed65VUkuY/giphy.gif",
	"https://media.giphy.com/media/26wkRxKJ9yUZzlorK/giphy.gif",
	"https://media.giphy.com/media/l46Cbqvg6gxGvh2PS/giphy.gif",
	"https://media.giphy.com/media/IlJ0FkaYggwkE/giphy.gif",
	"https://media.giphy.com/media/13sz48R33vovLi/giphy.gif",
	"https://media.giphy.com/media/Vj2fBk4JWGdxu/giphy.gif",
	"https://media.giphy.com/media/ict1QSd2CrvFe/giphy.gif",
	"https://media.giphy.com/media/3o752hpmTcQYvUsUmc/giphy.gif",
	"https://media.giphy.com/media/5Szs80FJTKDHQmA1SD/giphy.gif",
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
	channels []string
}

// NewOhMonday creates a new instance of the OhMonday plugin
func NewOhMonday(c *config.PluginConfig) (o *OhMonday, err error) {
	c.SetDefault(atTimeKey, defaultAtTime)

	scheduleDefinition := schedule.Definition{Interval: 1, Unit: schedule.Weeks, Weekday: time.Monday.String(), AtTime: c.GetString(atTimeKey)}

	o = new(OhMonday)
	o.Name = OhMondayPluginName
	o.channels = c.GetStringSlice(ohMondayChannelIDsKey)
	o.ScheduledActions = []slackscot.ScheduledActionDefinition{{Schedule: scheduleDefinition, Description: "Start the week off with a nice greeting", Action: o.sendGreeting}}

	return o, nil
}

func (o *OhMonday) sendGreeting(sender slackscot.RealTimeMessageSender) {
	for _, c := range o.channels {
		message := mondayPictures[selectionRandom.Intn(len(mondayPictures))]
		o.Logger.Debugf("[%s] Sending morning greeting message [%s] to [%s]", OhMondayPluginName, message, c)

		sender.SendNewMessage(message, c)
	}
}
