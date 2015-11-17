package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
)

type Configuration struct {
	Token       string                       `json:"token"`
	StoragePath string                       `json:"storagePath"`
	Extentions  map[string]map[string]string `json:"extensions"`
}

func LoadConfiguration(configurationPath string) (configuration *Configuration, err error) {
	content, err := ioutil.ReadFile(configurationPath)
	if err != nil {
		return nil, err
	}

	configuration = new(Configuration)
	err = json.Unmarshal(content, configuration)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to decode configuration file [%s]: %v", configurationPath, err))
	}

	return configuration, nil
}
