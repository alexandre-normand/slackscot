// Package config provides some utilities and structs to access configuration loaded via Viper
package config

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"time"
)

// Slackscot global configuration keys
const (
	TokenKey                    = "token"                                  // Slack token, string
	DebugKey                    = "debug"                                  // Debug mode, boolean
	MaxAgeHandledMessages       = "maxAgeHandledMessages"                  // The maximum age of messages before they are ignored (applicable for message updates)
	ResponseCacheSizeKey        = "responseCacheSize"                      // Response cache size in number of entries, int
	TimeLocationKey             = "timeLocation"                           // Time Location as understood by time.LoadLocation
	ThreadedRepliesKey          = "replyBehavior.threadedReplies"          // Threaded replies mode (slackscot will respond to all triggering messages using threads), boolean
	BroadcastThreadedRepliesKey = "replyBehavior.broadcastThreadedReplies" // Broadcast threaded replies (slackscot will set broadcast on threaded replies, only applies if threaded replies are enabled), boolean
	PluginsKey                  = "plugins"                                // Root element of the map of string key/values for plugins string
	UserInfoCacheSizeKey        = "userInfoCacheSize"                      // The number of entries to keep in the user info cache, int value. Defaults to no caching (value of 0)
)

// Advanced configuration keys, only change if you really know what you're doing and have reviewed the internals
const (
	MessageProcessingPartitionCount       = "advanced.messageProcessingPartitionCount"       // The number of partitions used to process messages concurrently. A higher number means less chance (but higher resource usage) of delays in processing of different messages
	MessageProcessingBufferedMessageCount = "advanced.messageProcessingBufferedMessageCount" // The channel capacity of each message processing partition. A higher number means less chance of a given partition with a lot of slow messages to handle causing a blockage of processing on other partitions
)

// Configuration defaults
const (
	debugDefault                             = false
	responseCacheSizeDefault                 = 5000
	timeLocationDefault                      = "Local"
	threadedRepliesDefault                   = false
	broadcastThreadedRepliesDefault          = false
	maxAgeHandledMessagesDefault             = time.Duration(24) * time.Hour
	msgProcessingPartitionCountDefault       = 1
	msgProcessingBufferedMessageCountDefault = 10
)

// ReplyBehavior holds flags to define the replying behavior (use threads or not and broadcast replies or not)
type ReplyBehavior struct {
	ThreadedReplies bool
	Broadcast       bool
}

// PluginConfig is a sub-viper instance holding the subtree specific to a named plugin
type PluginConfig = viper.Viper

// NewViperWithDefaults creates a new viper instance with defaults set on it
func NewViperWithDefaults() (v *viper.Viper) {
	v = viper.New()
	v.SetDefault(DebugKey, debugDefault)
	v.SetDefault(ResponseCacheSizeKey, responseCacheSizeDefault)
	v.SetDefault(TimeLocationKey, timeLocationDefault)
	v.SetDefault(ThreadedRepliesKey, threadedRepliesDefault)
	v.SetDefault(BroadcastThreadedRepliesKey, broadcastThreadedRepliesDefault)
	v.SetDefault(MaxAgeHandledMessages, maxAgeHandledMessagesDefault)
	v.SetDefault(MessageProcessingPartitionCount, msgProcessingPartitionCountDefault)
	v.SetDefault(MessageProcessingBufferedMessageCount, msgProcessingBufferedMessageCountDefault)

	return v
}

// GetTimeLocation reads the TimeLocation configuration and maps it to the appropriate time.Location value. Returns an err if the location value is invalid
func GetTimeLocation(v *viper.Viper) (timeLoc *time.Location, err error) {
	locationId := v.GetString(TimeLocationKey)

	// Load time zone location
	timeLoc, err = time.LoadLocation(locationId)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to load time zone location defined in configuration [%s]: [%s]", TimeLocationKey, locationId)
	}

	return timeLoc, nil
}

// GetPluginConfig returns the viper sub-tree for a named plugin
func GetPluginConfig(v *viper.Viper, name string) (pluginConfig *PluginConfig, err error) {
	pluginConfigPath := fmt.Sprintf("%s.%s", PluginsKey, name)
	if ok := v.IsSet(pluginConfigPath); !ok {
		return nil, fmt.Errorf("Missing plugin configuration for plugin [%s] at [%s]", name, pluginConfigPath)
	}

	subViper := v.Sub(pluginConfigPath)
	pc := PluginConfig(*subViper)
	return &pc, nil
}
