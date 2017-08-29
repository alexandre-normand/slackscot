Slackscot
=========

[![Go Report Card](https://goreportcard.com/badge/github.com/alexandre-normand/slackscot)](https://goreportcard.com/report/github.com/alexandre-normand/slackscot)
[![GoDoc](https://godoc.org/github.com/alexandre-normand/slackscot?status.svg)](https://godoc.org/github.com/alexandre-normand/slackscot)
[![Build Status](https://travis-ci.org/alexandre-normand/slackscot.svg)](https://travis-ci.org/alexandre-normand/slackscot) 

Slackscot is a simple [slack](https://slack.com) bot written in Go. It uses [Norberto Lopes](https://github.com/nlopes)'s [Slack API Integration](https://github.com/nlopes/slack) found at [https://github.com/nlopes/slack](https://github.com/nlopes/slack). The core functionality of the bot is previously used [James Bowman](https://github.com/james-bowman)'s [Slack RTM API integration](https://github.com/james-bowman/slack) and was heavily inspired by [talbot](https://github.com/james-bowman/talbot), also written by [James Bowman](https://github.com/james-bowman). 

The Name
--------
The first concrete bot implementation using this code was [youppi](https://github.com/alexandre-normand/youppi), named after the [great mascot](https://en.wikipedia.org/wiki/Youppi!) of the Montreal Expos and, when the Expos left Montreal, the Montreal Canadiens. `Slackscot` is a variation on the expected theme of `slackbot` with the implication that this is the core to _more_ than just a regular `bot`. You know, a friendly company mascot that hangs out on your `slack`. 

Features
--------

* Simple store API for persistence. It's basic a basic string key/string value thing.
* Basic config interface with slack token and storage path. 
* Plugin interface that is a logical grouping of one or many commands and "hear actions" (listeners). 

Concepts
--------

* Commands: commands are well-defined actions with a format. `Slackscot` handles all direct messages as implicit commands as well as `@mention <command>` on channels. Responses to commands are directed to the person who
  invoked it.
* Hear actions: those are listeners that can potentially match on any message sent on channels that `slackscot` is a member of. This can include actions that will randomly generate a response. Note that responses
  are not automatically directed to the person who authored the message triggering the response (although an implementation is free to use the user id of the triggering message if desired). 

How to use
----------
Slackscot provides the pieces to make your mascot but you'll have to assemble them for him/her to come alive. 


Here's an example of how [Youppi](https://github.com/alexandre-normand/youppi) does it:
```
package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/brain"
	"github.com/alexandre-normand/slackscot/config"
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

	youppi := slackscot.NewSlackscot("youppi", []slackscot.Plugin{plugins.NewKarma(), plugins.NewImager(), plugins.NewFingerQuoter(), plugins.NewEmojiBannerMaker()})

	err = youppi.Run(*config)
	if err != nil {
		log.Fatal(err)
	}
}
```

You'll also need to define your `json` configuration for the core, built-in extensions and any configuration required by your own custom extensions (not shown here):

```
{
   "token": "your-slack-bot-token",
   "debug": false,
   "storagePath": "/your-path-to-bot-home",
   "plugins": {
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
