[![Build Status](https://travis-ci.org/alexandre-normand/slackscot.svg)](https://travis-ci.org/alexandre-normand/slackscot)
[![GoDoc](https://godoc.org/github.com/alexandre-normand/slackscot?status.svg)](https://godoc.org/github.com/alexandre-normand/slackscot)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexandre-normand/slackscot)](https://goreportcard.com/report/github.com/alexandre-normand/slackscot)
[![Test Coverage](https://api.codeclimate.com/v1/badges/9fe1722ab2f31036c44c/test_coverage)](https://codeclimate.com/github/alexandre-normand/slackscot/test_coverage)
[![Maintainability](https://api.codeclimate.com/v1/badges/9fe1722ab2f31036c44c/maintainability)](https://codeclimate.com/github/alexandre-normand/slackscot/maintainability)

![logo.svg](logo.svg)

![name.svg](name.svg)

<!-- MarkdownTOC -->
- [Overview](#overview)
- [Requirements](#requirements)
- [Features](#features)
- [Demo](#demo)
- [The Name](#the-name)
- [Concepts](#concepts)
- [How to Use](#how-to-use)
	- [Integration and bringing your `slackscot` to life](#integration-and-bringing-your-slackscot-to-life)
	- [Configuration Example](#configuration-example)
	- [Creating Your Own Plugins](#creating-your-own-plugins)
- [Contributing](#contributing)
- [Some Credits](#some-credits)
<!-- /MarkdownTOC -->

# Overview 

`Slackscot` is a [slack](https://slack.com) `bot` core written in [Go](golang.org). 
Think of it as the assembly kit to making your own friendly `slack` `bot`. It comes
with a set of plugins you might enjoy and a friendly `API` for you to realize
your ambitious dreams (if you dreams include this sort of thing).

# Requirements

`Go 1.12` or above, mostly for [go module support](https://github.com/golang/go/wiki/Modules). 

# Features 

*   Support for reactions to message updates. `slackscot` does the following:
    *   Keeps track of plugin action responses and the message that triggered
        them

    *   On message updates:
        1.  Update responses for each triggered action

        2.  Delete responses that aren't triggering anymore (or result in 
            errors during the message update)

    *   On deletion of triggering messages, responses are also deleted

    *   *Limitation*: Sending a `message` automatically splits it into 
        multiple slack messages when it's too long. When updating messages,
	    this spitting doesn't happen and results in an `message too long` 
	    error. Effectively, the last message in the initial response might get
	    `deleted` as a result. Handling of this could be better but that is 
	    the current limitation 😕

*   Support for threaded replies to user message with option to also 
    `broadcast` on channels (disabled by `default`). 
    See [configuration example](#configuration-example) below where 
    both are enabled. 

*   Simple storage `API` for persistence. It's a basic `key:value` map backed
    by a filed-based [leveldb](https://github.com/syndtr/goleveldb)

*   Support for various configuration sources/formats via 
    [viper](https://github.com/spf13/viper)

*   Support for various ways to implement functionality: 
    1.  `scheduled actions`: run something every second, minute, hour, week. 
        [Oh Monday](plugins/ohmonday.go) is a plugin that demos this by 
        sending a `Monday` greeting every Monday at 10am (or the time you 
        configure it to).
    2.  `commands`: respond to a _command_ directed at your `slackscot`. That 
        means something like `@slackscot help` or a direct message `help`
        sent to `slackscot`.
    3.  `hear actions` (aka "listeners"): actions that evaluated for a match
        on every message that `slackscot` hears. You'll want to make sure
        your `Match` function isn't too heavy. An example is the "famous" 
        [finger quoter plugin](plugin/fingerquoter.go)

*   Built-in `help` plugin supporting a decently formatted help message
    as a command listing all plugins' actions. If you'd like some actions 
    to not be shown in the help, you can set `Hidden` to `true` in 
    its `ActionDefinition` (especially useful for `hear actions`)

*   The plugin interface as a logical grouping of one or many `commands` and 
    `hear actions` and/or `scheduled actions` 

*   Support for injected services providing plugins easy access to an optionally caching 
    `user info` and a `logger`. 

# Demo

*   `slackscot` deleting a triggered reaction after seeing a message updated 
    that caused the  first action to *not* trigger anymore and a new action 
    to now trigger (it makes sense when you see it)

![different-action-triggered-on-message-update](https://www.dropbox.com/s/r325jn1sqb7a93g/slackscot-update-message-trigger-different-action.gif?raw=1)

*   `slackscot` updating a triggered reaction after seeing a triggering message 
    being updated

![same-action-answer-update-on-message-update](https://www.dropbox.com/s/ge8p38bl977ugld/slackscot-update-same-action.gif?raw=1)

*   `slackscot` deleting a reaction after seeing the triggering message being
    deleted

![reaction-deletion-on-message-delete](https://www.dropbox.com/s/bwhq20m1b2obvx6/slackscot-delete-message-delete-answers.gif?raw=1)

# The Name

The first concrete bot implementation using this code was 
[youppi](https://github.com/alexandre-normand/youppi), named after the 
[great mascot](https://en.wikipedia.org/wiki/Youppi!) of the Montreal Expos 
and, when the Expos left Montreal, the Montreal Canadiens. 

`Slackscot` is a variation on the expected theme of *slackbot* with the 
implication that this is the core to *more* than just a regular `bot`. 
You know, a friendly company *mascot* that hangs out on your `slack`. 

# Concepts

*   `Commands`: commands are well-defined actions with a format. `Slackscot` 
    handles all direct messages as implicit commands as well as 
    `@mention <command>` on channels. Responses to commands are directed 
    to the person who invoked it.

*   `Hear actions`: those are listeners that can potentially match on any 
    message sent on channels that `slackscot` is a member of. This can 
    include actions that will randomly generate a response. Note that 
    responses are not automatically directed to the person who authored 
    the message triggering the response (although an implementation is 
    free to use the user id of the triggering message if desired). 

# How to Use

`Slackscot` provides the pieces to make your mascot but you'll have to 
assemble them for him/her to come alive. The easiest to get started is
to look at a real example: [youppi](https://github.com/alexandre-normand/youppi).

## Integration and bringing your `slackscot` to life

Here's an example of how [youppi](https://github.com/alexandre-normand/youppi) 
does it (apologies for the verbose and repetitive error handling when 
creating instances of plugins but it looks worse than it is):

```go
package main

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/spf13/viper"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"os"
)

var (
	configurationPath = kingpin.Flag("configuration", "The path to the configuration file.").Required().String()
	logfile           = kingpin.Flag("log", "The path to the log file").OpenFile(os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
)

func main() {
	kingpin.Version(VERSION)
	kingpin.Parse()

	v := config.NewViperWithDefaults()
	v.SetConfigFile(*configurationPath)
	err := v.ReadInConfig()
	if err != nil {
		log.Fatalf("Error loading configuration file [%s]: %v", *configurationPath, err)
	}

	// Do this only so that we can get a global debug flag available to everything
	viper.Set(config.DebugKey, v.GetBool(config.DebugKey))

	options := make([]slackscot.Option, 0)
	if *logfile != nil {
		options = append(options, slackscot.OptionLogfile(*logfile))
	}

	youppi, err := slackscot.NewSlackscot("youppi", v, options...)
	if err != nil {
		log.Fatal(err)
	}

	karma, err := plugins.NewKarma(v)
	if err != nil {
		log.Fatalf("Error initializing karma plugin: %v", err)
	}
	defer karma.Close()
	youppi.RegisterPlugin(&karma.Plugin)

	fingerQuoterConf, err := config.GetPluginConfig(v, plugins.FingerQuoterPluginName)
	if err != nil {
		log.Fatalf("Error initializing finger quoter plugin: %v", err)
	}
	fingerQuoter, err := plugins.NewFingerQuoter(fingerQuoterConf)
	if err != nil {
		log.Fatalf("Error initializing finger quoter plugin: %v", err)
	}
	youppi.RegisterPlugin(&fingerQuoter.Plugin)

	imager := plugins.NewImager()
	youppi.RegisterPlugin(&imager.Plugin)

	emojiBannerConf, err := config.GetPluginConfig(v, plugins.EmojiBannerPluginName)
	if err != nil {
		log.Fatalf("Error initializing emoji banner plugin: %v", err)
	}
	emojiBanner, err := plugins.NewEmojiBannerMaker(emojiBannerConf)
	if err != nil {
		log.Fatalf("Error initializing emoji banner plugin: %v", err)
	}
	defer emojiBanner.Close()
	youppi.RegisterPlugin(&emojiBanner.Plugin)

	ohMondayConf, err := config.GetPluginConfig(v, plugins.OhMondayPluginName)
	if err != nil {
		log.Fatalf("Error initializing oh monday plugin: %v", err)
	}
	ohMonday, err := plugins.NewOhMonday(ohMondayConf)
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

## Configuration Example

You'll also need to define your configuration for the `core`, used 
built-in plugins and any configuration required by your own custom plugins 
(not shown here). `Slackscot` uses 
[viper](https://github.com/spf13/viper) for loading configuration
which means that you are free to use a different file format 
(`yaml`, `toml`, `env variables`, etc.) as desired. 

```json
{
   "token": "your-slack-bot-token",
   "debug": false,
   "responseCacheSize": 5000,
   "timeLocation": "America/Los_Angeles",
   "storagePath": "/your-path-to-bot-home",
   "replyBehavior": {
      "threadedReplies": true,
      "broadcast": true
   },
   "plugins": {
      "ohMonday": {
   	     "channelId": "slackChannelId"
      },
      "fingerQuoter": {
         "frequency": "100",
         "channelIds": ""
      },
      "emojiBanner": {
         "figletFontUrl": "http://www.figlet.org/fonts/banner.flf"
      }
   }
}
```

## Creating Your Own Plugins

It might be best to look at examples in this repo to guide you through it:

*   The simplest plugin with a single `command` is the [versioner](plugins/versioner.go)
*   One example of `scheduled actions` is [oh monday](plugins/ohmonday.go)
*   One example of a mix of `hear actions` / `commands` that also uses the
    `store` api for persistence is the [karma](plugins/karma.go)

# Contributing

1.   Fork it (preferrably, outside the `GOPATH` as per the new 
     [go modules guidelines](https://github.com/golang/go/wiki/Modules#how-to-install-and-activate-module-support))
2.   Make your changes, commit them (don't forget to `go build ./...` and 
     `go test ./...`) and push your branch to your fork
3.   Open a PR and fill in the template (you don't have to but I'd appreciate context)
4.   Check the `code climate` and `travis` PR builds. You might have to fix things and
     there's no shame if you do. I probably won't merge something that doesn't pass
     `CI` build but I'm willing to help to get it to pass 🖖.  

# Some Credits
`slackscot` uses [Norberto Lopes](https://github.com/nlopes)'s 
[Slack API Integration](https://github.com/nlopes/slack) found at 
[https://github.com/nlopes/slack](https://github.com/nlopes/slack). The core
functionality of the bot is previously used 
[James Bowman](https://github.com/james-bowman)'s 
[Slack RTM API integration](https://github.com/james-bowman/slack) and 
was heavily inspired by [talbot](https://github.com/james-bowman/talbot), 
also written by [James Bowman](https://github.com/james-bowman). 
