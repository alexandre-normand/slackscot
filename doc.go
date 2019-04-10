/*
Package slackscot provides the building blocks to create a slack bot.

It is easily extendable via plugins that can combine commands, hear actions (listeners) as well
as scheduled actions. It also supports updating of triggered responses on message updates as well
as deleting triggered responses when the triggering messages are deleted by users.

Plugins also have access to services injected on startup by slackscot such as:
 - UserInfoFinder: To query user info
 - SLogger: To log debug/info statements
 - EmojiReactor: To emoji react to messages
 - FileUploader: To upload files
 - RealTimeMessageSender: To send unmanaged real time messages outside the normal reaction flow (i.e. for sending many messages or sending via a scheduled action)

Example code (from https://github.com/alexandre-normand/youppi):

	package main

	import (
		"github.com/alexandre-normand/slackscot"
		"github.com/alexandre-normand/slackscot/config"
		"github.com/alexandre-normand/slackscot/plugins"
 		"gopkg.in/alecthomas/kingpin.v2"
 		"io"
	)

	func main() {
		// TODO: Parse command-line, initialize viper and instantiate Storer implementation for needed for some plugins

		youppi, err := slackscot.NewBot("youppi", v, options...).
			WithPlugin(plugins.NewKarma(karmaStorer)).
			WithPlugin(plugins.NewTriggerer(triggererStorer)).
			WithConfigurablePluginErr(plugins.FingerQuoterPluginName, func(conf *config.PluginConfig) (p *slackscot.Plugin, err) { return plugins.NewFingerQuoter(c) }).
			WithConfigurablePluginCloserErr(plugins.EmojiBannerPluginName, func(conf *config.PluginConfig) (c io.Closer, p *slackscot.Plugin, err) { return plugins.NewEmojiBannerMaker(c) }).
			WithConfigurablePluginErr(plugins.OhMondayPluginName, func(conf *config.PluginConfig) (p *slackscot.Plugin, err) { return plugins.NewOhMonday(c) }).
			WithPlugin(plugins.NewVersionner(name, version)).
			Build()
		defer youppi.Close()

		if err != nil {
			log.Fatal(err)
		}

		err = youppi.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
*/
package slackscot
