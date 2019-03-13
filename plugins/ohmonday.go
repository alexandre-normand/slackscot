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
	"https://media.giphy.com/media/tvgUcaz62HqbC/giphy.gif",
	"https://media.giphy.com/media/Ytz5fkp09VIyc/giphy.gif",
	"https://media.giphy.com/media/3o7TKoktIaJdCiY1Ms/giphy.gif",
	"https://media.giphy.com/media/3D1v8iexqiPbq/giphy.gif",
	"https://media.giphy.com/media/f9RIpuEitaLuiczwFs/giphy.gif",
	"https://media.giphy.com/media/c6DIpCp1922KQ/giphy.gif",
	"https://media.giphy.com/media/3o6Zt4geeudbh6XxAs/giphy.gif",
	"https://media.giphy.com/media/kYq5TduyLd2YE/giphy.gif",
	"https://media.giphy.com/media/nsQpRYAvOn1eg/giphy.gif",
	"https://media.giphy.com/media/SAY0JN07b9yXC/giphy.gif",
	"https://media.giphy.com/media/jxTcPTeGxc5geNTzgU/giphy.gif",
	"https://media.giphy.com/media/d2Z7xYpg6eV2wAAU/giphy.gif",
	"https://media.giphy.com/media/pOKrXLf9N5g76/giphy.gif",
	"https://media.giphy.com/media/69kTTpTRc2t7GU1rKX/200w_d.gif",
	"https://media.giphy.com/media/l4FGG8qUJNxX6UJhK/giphy.gif",
	"https://media.giphy.com/media/PhBf5O2mPItJm/giphy.gif",
	"https://66.media.tumblr.com/0d8e767123ebd3b1cf2870ed0433a4a0/tumblr_inline_odd3fcGwAw1raprkq_400.gif",
	"https://66.media.tumblr.com/c2b03e242ea5cd2b7b3f92de9a60b32a/tumblr_inline_odd3a4Zm4S1raprkq_500.gif",
	"https://66.media.tumblr.com/e0e87cf77dfecb79d78afb94f12d3b17/tumblr_inline_oauhktAydg1raprkq_400.gif",
	"https://66.media.tumblr.com/2d334e6df80465834cc410edc5f7fbc4/tumblr_inline_oa9i9maU1C1raprkq_400.gif",
	"https://66.media.tumblr.com/94185caa6fa578cdf2492e62cb0666ab/tumblr_inline_o91sjrJiGK1raprkq_400.gif",
	"https://66.media.tumblr.com/3edfac3344c0d902e10dc36a293bb9d9/tumblr_inline_o91sbhsOqL1raprkq_400.gif",
	"https://66.media.tumblr.com/70c33cf96cdaced05c98c282186f79c8/tumblr_inline_o8bpkmLHEt1raprkq_400.gif",
	"https://66.media.tumblr.com/9f9e82d2796baa356cc2b6bfa5e8b28e/tumblr_inline_o8bp9qj4DY1raprkq_400.gif",
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

		sender.SendNewMessage(c, message)
	}
}
