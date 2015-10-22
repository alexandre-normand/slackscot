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
func main() {
	kingpin.Parse()

	config, err := config.LoadConfiguration(*configurationPath)
	if err != nil {
		log.Fatal(err)
	}

	var youppi Youppi

	// Registers deferred call to close resources on shutdown
	defer youppi.Close()

	slackscot.Run(youppi, *config)
}

type Youppi struct {
	bundles []slackscot.ExtentionBundle
}

func (youppi Youppi) Init(config config.Configuration) (commands []slackscot.Action, listeners []slackscot.Action, err error) {
	karma := brain.NewKarma()
	c, l, err := karma.Init(config)
	if err != nil {
		return nil, nil, err
	}

	commands = append(commands, c...)
	listeners = append(listeners, l...)

	imager := brain.NewImager()
	c, l, err = imager.Init(config)
	if err != nil {
		return nil, nil, err
	}
	commands = append(commands, c...)
	listeners = append(listeners, l...)

	//initImagesExt(config)
	//initServiceCheck(config)
	return commands, listeners, nil

}

func (youppi Youppi) Close() {
	// Close any resources needed by scripts
	for _, b := range youppi.bundles {
		b.Close()
	}
}

```
