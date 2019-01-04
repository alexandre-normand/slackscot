[![Go Report Card](https://goreportcard.com/badge/github.com/alexandre-normand/slackscot)](https://goreportcard.com/report/github.com/alexandre-normand/slackscot)
[![GoDoc](https://godoc.org/github.com/alexandre-normand/slackscot?status.svg)](https://godoc.org/github.com/alexandre-normand/slackscot)
[![Build Status](https://travis-ci.org/alexandre-normand/slackscot.svg)](https://travis-ci.org/alexandre-normand/slackscot)

![logo.svg](logo.svg)

![name.svg](name.svg)

`Slackscot` is a [slack](https://slack.com) bot written in Go. It uses [Norberto Lopes](https://github.com/nlopes)'s [Slack API Integration](https://github.com/nlopes/slack) found at [https://github.com/nlopes/slack](https://github.com/nlopes/slack). The core functionality of the bot is previously used [James Bowman](https://github.com/james-bowman)'s [Slack RTM API integration](https://github.com/james-bowman/slack) and was heavily inspired by [talbot](https://github.com/james-bowman/talbot), also written by [James Bowman](https://github.com/james-bowman). 

The Name
--------
The first concrete bot implementation using this code was [youppi](https://github.com/alexandre-normand/youppi), named after the [great mascot](https://en.wikipedia.org/wiki/Youppi!) of the Montreal Expos and, when the Expos left Montreal, the Montreal Canadiens. 

`Slackscot` is a variation on the expected theme of _slackbot_ with the implication that this is the core to _more_ than just a regular `bot`. You know, a friendly company _mascot_ that hangs out on your `slack`. 

Features
--------

* Simple store API for persistence. It's basic a basic string key/string value thing.
* Basic config interface with slack token and storage path. 
* Plugin interface that is a logical grouping of one or many commands and "hear actions" (listeners). 
* Support for various configuration sources/formats via [viper](https://github.com/spf13/viper)

### Fancy Features

* Support for reactions to message updates. `slackscot` does the following:
	- Keeps track of plugin action responses and the message that triggered them
	- On message updates:
		1. Update responses for each triggered action
		2. Delete responses that aren't triggering anymore (or result in errors during the message update)
	- On deletion of triggering messages, responses are also deleted
	- *Limitation*: Sending a `message` automatically splits it into multiple slack messages when it's too long. When updating messages,
	  this spitting doesn't happen and results in an `message too long` error. Effectively, the last message in the initial response might get
	  `deleted` as a result. Handling of this could be better but that is the current limitation.
* Support for threaded replies to user message with option to also `broadcast` on channels (disabled by `default`). See [configuration example](#configuration-example) below where both are enabled. 
* Support for scheduled actions

Concepts
--------

* Commands: commands are well-defined actions with a format. `Slackscot` handles all direct messages as implicit commands as well as `@mention <command>` on channels. Responses to commands are directed to the person who
  invoked it.
* Hear actions: those are listeners that can potentially match on any message sent on channels that `slackscot` is a member of. This can include actions that will randomly generate a response. Note that responses
  are not automatically directed to the person who authored the message triggering the response (although an implementation is free to use the user id of the triggering message if desired). 

How to use
----------
Slackscot provides the pieces to make your mascot but you'll have to assemble them for him/her to come alive. 

### Integration and bringing your `slackscot` to life

Here's an example of how [Youppi](https://github.com/alexandre-normand/youppi) does it (apologies for the verbose and repetitive error handling when creating instances of plugins):
```
package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/plugins"
	"log"
)

var (
	configurationPath = kingpin.Flag("configuration", "The path to the configuration file.").Required().String()
)

func main() {
	kingpin.Parse()

	config, err := config.LoadConfiguration(*configurationPath)
	if err != nil {
		log.Fatal(err)
	}

	c := *config

	youppi, err := slackscot.NewSlackscot("youppi", c)
	if err != nil {
		log.Fatal(err)
	}

	karma, err := plugins.NewKarma(c)
	if err != nil {
		log.Fatalf("Error initializing karma plugin: %v", err)
	}
	defer karma.Close()
	youppi.RegisterPlugin(&karma.Plugin)

	fingerQuoter, err := plugins.NewFingerQuoter(c)
	if err != nil {
		log.Fatalf("Error initializing finger quoter plugin: %v", err)
	}
	youppi.RegisterPlugin(&fingerQuoter.Plugin)

	imager := plugins.NewImager()
	youppi.RegisterPlugin(&imager.Plugin)

	emojiBanner, err := plugins.NewEmojiBannerMaker(c)
	if err != nil {
		log.Fatalf("Error initializing emoji banner plugin: %v", err)
	}
	youppi.RegisterPlugin(&emojiBanner.Plugin)

	ohMonday, err := plugins.NewOhMonday(c)
	if err != nil {
		log.Fatalf("Error initializing oh monday plugin: %v", err)
	}
	youppi.RegisterPlugin(&ohMonday.Plugin)

	versioner := plugins.NewVersioner("youppi", VERSION)
	youppi.RegisterPlugin(&versioner.Plugin)

	err = youppi.Run()
	if err != nil {
		log.Fatal(err)
	}
}

```

### Configuration example

You'll also need to define your configuration for the `core`, built-in extensions and any configuration required by your own custom extensions (not shown here). `Slackscot` uses [viper](https://github.com/spf13/viper) for loading configuration
which means that you are free to use a different file format (`yaml`, `toml`, etc.) as desired. 

```
{
   "token": "your-slack-bot-token",
   "debug": false,
   "responseCacheSize": 5000,
   "timeLocation": "America/Los_Angeles",
   "storagePath": "/your-path-to-bot-home",
   "replyBehavior": {
      "threadedReplies": true,
      "broadcast": true
   }
   "plugins": {
   	  "ohMonday": {
   	  	 "channelId": "slackChannelId"
   	  },
      "fingerQuoter": {
         "frequency": "100",
         "channelIds": ""
      },
      "emojiBanner": {
         "fontPath": "/your-path-to-bot-home/fonts",
         "fontName": "banner"
      }
   }
}
```
