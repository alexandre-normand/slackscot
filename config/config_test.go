package config_test

import (
	"github.com/alexandre-normand/slackscot/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewWithDefault(t *testing.T) {
	v := config.NewViperWithDefaults()

	assert.Equal(t, false, v.GetBool(config.DebugKey), "%s should be %t", config.DebugKey, false)
	assert.Equal(t, 5000, v.GetInt(config.ResponseCacheSizeKey), "%s should be %d", config.ResponseCacheSizeKey, 5000)
	assert.Equal(t, "Local", v.GetString(config.TimeLocationKey), "%s should be %s", config.TimeLocationKey, "Local")
	assert.Equal(t, false, v.GetBool(config.ThreadedRepliesKey), "%s should be %t", config.ThreadedRepliesKey, false)
	assert.Equal(t, true, v.GetBool(config.BroadcastThreadedRepliesKey), "%s should be %t", config.BroadcastThreadedRepliesKey, true)
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
