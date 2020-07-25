package config_test

import (
	"github.com/alexandre-normand/slackscot/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewWithDefault(t *testing.T) {
	v := config.NewViperWithDefaults()

	assert.Equal(t, false, v.GetBool(config.DebugKey), "%s should be %t", config.DebugKey, false)
	assert.Equal(t, 5000, v.GetInt(config.ResponseCacheSizeKey), "%s should be %d", config.ResponseCacheSizeKey, 5000)
	assert.Equal(t, "Local", v.GetString(config.TimeLocationKey), "%s should be %s", config.TimeLocationKey, "Local")
	assert.Equal(t, false, v.GetBool(config.ThreadedRepliesKey), "%s should be %t", config.ThreadedRepliesKey, false)
	assert.Equal(t, false, v.GetBool(config.BroadcastThreadedRepliesKey), "%s should be %t", config.BroadcastThreadedRepliesKey, false)
	assert.Equal(t, time.Duration(24)*time.Hour, v.GetDuration(config.MaxAgeHandledMessages), "%s should be %t", config.MaxAgeHandledMessages, time.Duration(24)*time.Hour)
	assert.Equal(t, 16, v.GetInt(config.MessageProcessingPartitionCount), "%s should be %d", config.MessageProcessingPartitionCount, 16)
	assert.Equal(t, 10, v.GetInt(config.MessageProcessingBufferedMessageCount), "%s should be %d", config.MessageProcessingBufferedMessageCount, 10)
}

func TestLayerConfigWithDefaults(t *testing.T) {
	v := viper.New()

	for key := range config.NewViperWithDefaults().AllSettings() {
		assert.Nil(t, v.Get(key))
	}

	v = config.LayerConfigWithDefaults(v)
	for key, expectedVal := range config.NewViperWithDefaults().AllSettings() {
		assert.Equal(t, expectedVal, v.Get(key), "%s should be %v", key, expectedVal)
	}
}

func TestLayeredConfigWithDefaultsAndOverrides(t *testing.T) {
	v := viper.New()
	v = config.LayerConfigWithDefaults(v)
	v.Set(config.MessageProcessingPartitionCount, 32)
	v.Set(config.MessageProcessingBufferedMessageCount, 20)

	v = config.LayerConfigWithDefaults(v)
	for key, expectedVal := range config.NewViperWithDefaults().AllSettings() {
		if key != "advanced" {
			assert.Equal(t, expectedVal, v.Get(key), "%s should be %v", key, expectedVal)
		}
	}

	assert.Equal(t, 32, v.GetInt(config.MessageProcessingPartitionCount), "%s should be %v", config.MessageProcessingPartitionCount, 32)
	assert.Equal(t, 20, v.GetInt(config.MessageProcessingBufferedMessageCount), "%s should be %v", config.MessageProcessingBufferedMessageCount, 20)
}

func TestGetTimeLocationWithDefault(t *testing.T) {
	v := viper.New()
	v.Set(config.TimeLocationKey, "Local")

	timeLoc, err := config.GetTimeLocation(v)

	assert.Nil(t, err)
	if assert.NotNil(t, timeLoc) {
		assert.Conditionf(t, func() bool { return timeLoc.String() == "Local" || timeLoc.String() == "UTC" }, "timeLoc should be either Local or UTC but was %s", timeLoc.String())
	}
}

func TestGetTimeLocationWithTimezoneId(t *testing.T) {
	v := viper.New()
	v.Set(config.TimeLocationKey, "America/Los_Angeles")

	timeLoc, err := config.GetTimeLocation(v)

	assert.Nil(t, err)
	if assert.NotNil(t, timeLoc) {
		assert.Equal(t, "America/Los_Angeles", timeLoc.String())
	}
}

func TestGetTimeLocationWithInvalidValue(t *testing.T) {
	v := viper.New()
	v.Set(config.TimeLocationKey, "invalid")

	_, err := config.GetTimeLocation(v)

	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "invalid")
	}
}

func TestGetPluginConfig(t *testing.T) {
	v := viper.New()
	configValues := map[string]interface{}{
		"feature1": true,
		"subFeature": map[string]string{
			"name":  "John",
			"email": "test@golang.org",
		},
	}
	// Set the test configuration
	v.Set(config.PluginsKey, map[string]interface{}{
		"pluginName": configValues,
	})

	pc, err := config.GetPluginConfig(v, "pluginName")

	assert.Nil(t, err)
	if assert.NotNil(t, pc) {
		assert.Equal(t, configValues["subFeature"], pc.GetStringMapString("subFeature"))
	}
}

func TestGetPluginConfigWithMissingConfig(t *testing.T) {
	v := viper.New()

	_, err := config.GetPluginConfig(v, "pluginName")

	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "Missing plugin configuration for plugin [pluginName]")
	}
}
