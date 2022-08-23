package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ovh/configstore"
)

const (
	configAlias = "config"
)

type mqttConfig struct {
	Host     string  `json:"host"`
	Port     int     `json:"port"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
}

type topicsConfig struct {
	SensorFormat  string `json:"sensor_format"`
	HomeAssistant string `json:"home_assistant"`
}

type sensorConfig struct {
	Name       string `json:"name"`
	MacAddress string `json:"mac_address"`
}

type config struct {
	MQTT    mqttConfig      `json:"mqtt"`
	Topics  topicsConfig    `json:"topics"`
	Sensors []*sensorConfig `json:"sensors"`
}

func loadConfig(store *configstore.Store) (*config, error) {
	var notFound configstore.ErrItemNotFound

	itemFilter := configstore.Filter().Store(store).Slice(configAlias).Squash()
	configItem, err := itemFilter.GetFirstItem()
	if err != nil {
		if errors.As(err, &notFound) {
			return nil, fmt.Errorf("configstore: get %q: no item found", configAlias)
		}
		return nil, err
	}

	jsonConfig, err := configItem.Value()
	if err != nil {
		return nil, err
	}

	var cfg config

	return &cfg, json.Unmarshal([]byte(jsonConfig), &cfg)
}
