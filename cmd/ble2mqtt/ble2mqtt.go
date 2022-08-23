package main

import (
	"context"
	"encoding/binary"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-ble/ble"
	"github.com/ovh/configstore"
	"github.com/rclsilver/ble2mqtt"
	"github.com/sirupsen/logrus"
)

func main() {
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	configstore.InitFromEnvironment()

	cfg, err := loadConfig(configstore.DefaultStore)
	if err != nil {
		logrus.WithError(err).Fatal("error while loading configuration")
	}
	logrus.Debug("configuration loaded")

	bt, err := ble2mqtt.InitBluetooth()
	if err != nil {
		logrus.WithError(err).Fatal("error while initializing bluetooth")
	}
	logrus.Debug("bluetooth initialized")

	mqtt, err := ble2mqtt.InitMQTT(
		cfg.MQTT.Host,
		cfg.MQTT.Port,
		cfg.MQTT.Username,
		cfg.MQTT.Password,
	)
	if err != nil {
		logrus.WithError(err).Fatal("error while initializing MQTT")
	}
	logrus.Debug("MQTT initialized")

	if err := mqtt.Connect(); err != nil {
		logrus.WithError(err).Fatal("error while connecting to the MQTT broker")
	}

	sensors := make(map[string]*sensorConfig)
	for _, sensor := range cfg.Sensors {
		sensors[strings.ToLower(sensor.MacAddress)] = sensor
	}

	for {
		ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), time.Minute))

		if err := bt.Scan(ctx, func(a ble.Advertisement) {
			sensor, ok := sensors[strings.ToLower(a.Addr().String())]
			if !ok {
				return
			}

			for _, data := range a.ServiceData() {
				if len(data.Data) >= 15 {
					// 00:05 => mac
					// 06:07 => temperature
					// 08:09 => humidity
					// 10:11 => battery voltage
					//    12 => battery level
					//    13 => count
					//    14 => flag

					temperature := float64(binary.LittleEndian.Uint16(data.Data[6:])) / 100.0
					humidity := float64(binary.LittleEndian.Uint16(data.Data[8:])) / 100.0
					batteryLevel := data.Data[12]
					timestamp := time.Now().Unix()
					timestamp -= timestamp % 60

					logrus.WithFields(logrus.Fields{
						"temperature": temperature,
						"humidity":    humidity,
						"battery":     batteryLevel,
						"rssi":        a.RSSI(),
					}).Debugf("new advertisement from %s (%s)", sensor.Name, sensor.MacAddress)

					topic := strings.ReplaceAll(cfg.Topics.SensorFormat, "${name}", sensor.Name)

					temperatureTopic := strings.ReplaceAll(topic, "${sensor}", "temperature")
					temperaturePayload := map[string]interface{}{
						"timestamp": timestamp,
						"label":     strings.Title(sensor.Name),
						"value":     temperature,
						"battery":   int16(batteryLevel),
						"signal":    a.RSSI(),
					}
					if err := mqtt.Publish(temperatureTopic, temperaturePayload); err != nil {
						logrus.WithField("sensor", sensor.MacAddress).WithError(err).Error("error while publishing temperature payload")
					}

					humidityTopic := strings.ReplaceAll(topic, "${sensor}", "humidity")
					humidityPayload := map[string]interface{}{
						"timestamp": timestamp,
						"label":     strings.Title(sensor.Name),
						"value":     humidity,
						"battery":   int16(batteryLevel),
						"signal":    a.RSSI(),
					}
					if err := mqtt.Publish(humidityTopic, humidityPayload); err != nil {
						logrus.WithField("sensor", sensor.MacAddress).WithError(err).Error("error while publishing humidity payload")
					}

					haBaseTopic := cfg.Topics.HomeAssistant + "/sensor/" + strings.ReplaceAll(sensor.MacAddress, ":", "")
					haDevice := map[string]interface{}{
						"name":         "LYWSD03MMC " + sensor.Name,
						"identifiers":  sensor.MacAddress,
						"manufacturer": "Xiaomi",
						"model":        "LYWSD03MMC",
					}

					haTemperatureTopic := haBaseTopic + "/" + sensor.Name + "_temperature/config"
					haTemperaturePayload := map[string]interface{}{
						"device":              haDevice,
						"name":                sensor.Name + " - temperature",
						"unique_id":           sensor.Name + "_temperature",
						"state_topic":         temperatureTopic,
						"value_template":      "{{ value_json.value }}",
						"unit_of_measurement": "°C",
					}
					if err := mqtt.Publish(haTemperatureTopic, haTemperaturePayload); err != nil {
						logrus.WithField("sensor", sensor.MacAddress).WithError(err).Error("error while publishing temperature home-assistant discovery")
					}

					haHumidityTopic := haBaseTopic + "/" + sensor.Name + "_humidity/config"
					haHumidityPayload := map[string]interface{}{
						"device":              haDevice,
						"name":                sensor.Name + " - humidity",
						"unique_id":           sensor.Name + "_humidity",
						"state_topic":         humidityTopic,
						"value_template":      "{{ value_json.value }}",
						"unit_of_measurement": "%",
					}
					if err := mqtt.Publish(haHumidityTopic, haHumidityPayload); err != nil {
						logrus.WithField("sensor", sensor.MacAddress).WithError(err).Error("error while publishing humidity home-assistant discovery")
					}

					haBatteryTopic := haBaseTopic + "/" + sensor.Name + "_battery/config"
					haBatteryPayload := map[string]interface{}{
						"device":              haDevice,
						"name":                sensor.Name + " - battery",
						"unique_id":           sensor.Name + "_battery",
						"device_class":        "battery",
						"state_topic":         temperatureTopic,
						"value_template":      "{{ value_json.battery }}",
						"unit_of_measurement": "%",
					}
					if err := mqtt.Publish(haBatteryTopic, haBatteryPayload); err != nil {
						logrus.WithField("sensor", sensor.MacAddress).WithError(err).Error("error while publishing battery home-assistant discovery")
					}
				}
			}
		}); err != nil {
			logrus.WithError(err).Fatal("error while scanning")
		}
	}

	// TODO
	//mqtt.Disconnect()
}
