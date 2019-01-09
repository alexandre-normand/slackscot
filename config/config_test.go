package config

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewWithDefault(t *testing.T) {
	v := NewViperWithDefaults()

	assert.Equal(t, debugDefault, v.GetBool(DebugKey), "%s should be %t", DebugKey, debugDefault)
	assert.Equal(t, responseCacheSizeDefault, v.GetInt(ResponseCacheSizeKey), "%s should be %d", ResponseCacheSizeKey, responseCacheSizeDefault)
	assert.Equal(t, timeLocationDefault, v.GetString(TimeLocationKey), "%s should be %s", TimeLocationKey, timeLocationDefault)
	assert.Equal(t, threadedRepliesDefault, v.GetBool(ThreadedRepliesKey), "%s should be %t", ThreadedRepliesKey, threadedRepliesDefault)
	assert.Equal(t, broadcastThreadedRepliesDefault, v.GetBool(BroadcastThreadedRepliesKey), "%s should be %t", BroadcastThreadedRepliesKey, broadcastThreadedRepliesDefault)
}

func TestGetTimeLocationWithDefault(t *testing.T) {
	v := viper.New()
	v.Set(TimeLocationKey, timeLocationDefault)

	timeLoc, err := GetTimeLocation(v)

	assert.Nil(t, err)
	if assert.NotNil(t, timeLoc) {
		assert.Equal(t, "Local", timeLoc.String())
	}
}

func TestGetTimeLocationWithTimezoneId(t *testing.T) {
	v := viper.New()
	v.Set(TimeLocationKey, "America/Los_Angeles")

	timeLoc, err := GetTimeLocation(v)

	assert.Nil(t, err)
	if assert.NotNil(t, timeLoc) {
		assert.Equal(t, "America/Los_Angeles", timeLoc.String())
	}
}

func TestGetTimeLocationWithInvalidValue(t *testing.T) {
	v := viper.New()
	v.Set(TimeLocationKey, "invalid")

	_, err := GetTimeLocation(v)

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
	v.Set(PluginsKey, map[string]interface{}{
		"pluginName": configValues,
	})

	pc, err := GetPluginConfig(v, "pluginName")

	assert.Nil(t, err)
	if assert.NotNil(t, pc) {
		assert.Equal(t, configValues["subFeature"], pc.GetStringMapString("subFeature"))
	}
}

func TestGetPluginConfigWithMissingConfig(t *testing.T) {
	v := viper.New()

	_, err := GetPluginConfig(v, "pluginName")

	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "Missing plugin configuration for plugin [pluginName]")
	}
}
