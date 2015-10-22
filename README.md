Slackscot
=========

[![GoDoc](https://godoc.org/github.com/alexandre-normand/slackscot?status.svg)](https://godoc.org/github.com/alexandre-normand/slackscot)
[![Build Status](https://travis-ci.org/alexandre-normand/slackscot.svg)](https://travis-ci.org/alexandre-normand/slackscot) 

Slackscot is a simple [slack](https://slack.com) bot written in Go. It uses [Slack RTM API Integration](https://github.com/james-bowman/slack) written by [James Bowman](https://github.com/james-bowman). The core functionality of the bot is also heavily inspired by [talbot](https://github.com/james-bowman/talbot), also written by [James Bowman](https://github.com/james-bowman). 

The Name
--------
The first concrete bot implementation using this code was [youppi](https://github.com/alexandre-normand/youppi), named after the [great mascot](https://en.wikipedia.org/wiki/Youppi!) of the Montreal Expos and, when the Expos left Montreal, the Montreal Canadiens. Slackscot is a play on words for `mascot` because who doesn't want a hairy colorful friend as company on slack? 

Features
--------

* Simple store API for persistence. It's basic a basic string key/string value thing.
* Basic config interface with slack token and storage path. 
* A few basic extension bundles in [brain](brain). 

How to use
----------
Slackscot provides the pieces to make your mascot but you'll have to assemble them for it to come alive. 


Here's an example of how Youppi does it:
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

	youppi := slackscot.NewSlackscot([]slackscot.ExtentionBundle{brain.NewKarma(), brain.NewImager()})

	slackscot.Run(*youppi, *config)
}
```
