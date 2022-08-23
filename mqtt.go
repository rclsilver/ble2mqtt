package ble2mqtt

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTT struct {
	client mqtt.Client
}

func InitMQTT(host string, port int, username, password *string) (*MQTT, error) {
	opts := mqtt.NewClientOptions()
	opts.SetClientID("ble2mqtt")
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", host, port))

	if username != nil {
		opts.SetUsername(*username)
	}

	if password != nil {
		opts.SetPassword(*password)
	}

	return &MQTT{
		client: mqtt.NewClient(opts),
	}, nil
}

func (m *MQTT) Connect() error {
	token := m.client.Connect()
	token.Wait()

	return token.Error()
}

func (m *MQTT) Publish(topic string, payload map[string]interface{}) error {
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	token := m.client.Publish(topic, 0, true, payloadJson)
	token.Wait()

	return token.Error()
}

func (m *MQTT) Disconnect() {
	m.client.Disconnect(0)
}
