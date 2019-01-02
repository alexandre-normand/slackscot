// Package config provides a method for loading a slackscot configuration from file
package config

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"time"
)

// Configuration holds a slackscot instance configuration
type Configuration struct {
	Token             string                       `json:"token"`
	Debug             bool                         `json:"debug"`
	ResponseCacheSize int                          `json:"responseCacheSize"`
	Location          string                       `json:"timeLocation"` // Optional time location (see https://golang.org/pkg/time/#Location.String). Default is "Local"
	ReplyBehavior     ReplyBehavior                `json:"replyBehavior"`
	StoragePath       string                       `json:"storagePath"`
	Plugins           map[string]map[string]string `json:"plugins"`
	TimeLocation      *time.Location
}

// ReplyBehavior holds flags to define the replying behavior (use threads or not and broadcast replies or not)
type ReplyBehavior struct {
	ThreadedReplies bool `json:"threadedReplies"`
	Broadcast       bool `json:"broadcast"`
}

var defaultConfiguration = Configuration{Debug: false, ResponseCacheSize: 5000, Location: "Local", ReplyBehavior: ReplyBehavior{ThreadedReplies: false, Broadcast: true}}

// LoadConfiguration loads a slackscot configuration from a given file path
func LoadConfiguration(configurationPath string) (configuration *Configuration, err error) {
	content, err := ioutil.ReadFile(configurationPath)
	if err != nil {
		return nil, err
	}

	c := defaultConfiguration
	err = json.Unmarshal(content, &c)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse configuration file [%s]: %v", configurationPath, err)
	}

	// Load time zone location
	c.TimeLocation, err = time.LoadLocation(c.Location)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to load time zone location defined in configuration [%s]: [%s]", "timeLocation", c.Location)
	}

	return &c, nil
}
