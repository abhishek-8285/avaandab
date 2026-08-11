import mqtt from 'mqtt';
import { getMQTTBrokerURL } from '../constants/network';

class MQTTTelemetryService {
  private client: mqtt.MqttClient | null = null;
  private isConnected = false;

  connect(driverId: string): void {
    try {
      const brokerUrl = getMQTTBrokerURL();
      console.log('[MQTT CONNECTING] Target Broker URL:', brokerUrl);

      // Connect to MQTT Broker over WebSockets
      this.client = mqtt.connect(brokerUrl, {
        clientId: `driver_${driverId}_${Math.random().toString(16).substring(2, 8)}`,
        keepalive: 60,
        reconnectPeriod: 5000,
      });

      this.client.on('connect', () => {
        this.isConnected = true;
        console.log('[MQTT MOBILE SUCCESS] Connected to MQTT Telemetry Broker');

        // Subscribe to driver dispatch assignment updates
        const updateTopic = `avandab/trips/drivers/${driverId}/updates`;
        this.client?.subscribe(updateTopic, (err) => {
          if (!err) {
            console.log(`[MQTT SUBSCRIBED] Listening on topic: ${updateTopic}`);
          }
        });
      });

      this.client.on('message', (topic, message) => {
        console.log(`[MQTT RECV] Topic: ${topic} Payload: ${message.toString()}`);
      });

      this.client.on('error', (err) => {
        console.log('[MQTT MOBILE WARNING] Connection warning (fallback to HTTP):', err.message);
      });
    } catch (e: any) {
      console.log('[MQTT INIT WARNING]', e.message);
    }
  }

  // Publish high-frequency live GPS coordinates over MQTT
  publishLocation(driverId: string, latitude: number, longitude: number): void {
    if (this.client && this.isConnected) {
      const topic = `avandab/telemetry/drivers/${driverId}/gps`;
      const payload = JSON.stringify({
        driver_id: driverId,
        latitude,
        longitude,
        timestamp: new Date().toISOString(),
      });
      this.client.publish(topic, payload, { qos: 1 });
      console.log(`[MQTT PUBLISHED GPS] Lat: ${latitude.toFixed(4)}, Lng: ${longitude.toFixed(4)} -> ${topic}`);
    }
  }

  disconnect(): void {
    if (this.client) {
      this.client.end();
      this.isConnected = false;
    }
  }
}

export const MQTT = new MQTTTelemetryService();
