package mqttservice

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type GPSTelemetryPayload struct {
	DriverID  string    `json:"driver_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}

type MQTTBroker struct {
	client mqtt.Client
}

func NewMQTTBroker(brokerURL string) *MQTTBroker {
	opts := mqtt.NewClientOptions().AddBroker(brokerURL)
	opts.SetClientID("avandab_backend_server")
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("[MQTT WARNING] Could not connect to MQTT Broker (%s): %v (Running fallback mode)", brokerURL, token.Error())
	} else {
		log.Println("[MQTT SUCCESS] Connected to MQTT Telemetry Broker at", brokerURL)
	}

	b := &MQTTBroker{client: client}
	b.subscribeTelemetry()
	return b
}

func (b *MQTTBroker) subscribeTelemetry() {
	if !b.client.IsConnected() {
		return
	}
	topic := "avandab/telemetry/drivers/+/gps"
	b.client.Subscribe(topic, 1, func(c mqtt.Client, m mqtt.Message) {
		var payload GPSTelemetryPayload
		if err := json.Unmarshal(m.Payload(), &payload); err == nil {
			log.Printf("[MQTT TELEMETRY RECV] Driver %s -> Lat: %.4f, Lng: %.4f", payload.DriverID, payload.Latitude, payload.Longitude)
		}
	})
}

func (b *MQTTBroker) PublishTripUpdate(driverID string, tripID string, status string) {
	if !b.client.IsConnected() {
		return
	}
	topic := fmt.Sprintf("avandab/trips/drivers/%s/updates", driverID)
	data, _ := json.Marshal(map[string]string{
		"trip_id": tripID,
		"status":  status,
		"time":    time.Now().Format(time.RFC3339),
	})
	b.client.Publish(topic, 1, false, data)
}
