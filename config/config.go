package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Configuration holds a slackscot instance configuration
type Configuration struct {
	Token             string                       `json:"token"`
	Debug             bool                         `json:"debug"`
	ResponseCacheSize int                          `json:"responseCacheSize"`
	StoragePath       string                       `json:"storagePath"`
	Plugins           map[string]map[string]string `json:"plugins"`
}

// LoadConfiguration loads a slackscot configuration from a given file path
func LoadConfiguration(configurationPath string) (configuration *Configuration, err error) {
	content, err := ioutil.ReadFile(configurationPath)
	if err != nil {
		return nil, err
	}

	configuration = new(Configuration)
	err = json.Unmarshal(content, configuration)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse configuration file [%s]: %v", configurationPath, err)
	}

	return configuration, nil
}
