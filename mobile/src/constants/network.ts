export const API_SCHEME = process.env.EXPO_PUBLIC_API_SCHEME || 'http';
export const MQTT_SCHEME = process.env.EXPO_PUBLIC_MQTT_SCHEME || 'ws';
export const BACKEND_HOST = process.env.EXPO_PUBLIC_BACKEND_HOST || '127.0.0.1';
export const API_PORT = Number(process.env.EXPO_PUBLIC_API_PORT || 8080);
export const MQTT_BROKER_PORT = Number(process.env.EXPO_PUBLIC_MQTT_BROKER_PORT || 9001);

export const API_BASE_URL = `${API_SCHEME}://${BACKEND_HOST}:${API_PORT}`;
export const MQTT_BROKER_URL = `${MQTT_SCHEME}://${BACKEND_HOST}:${MQTT_BROKER_PORT}`;

// Demo fallback values used only when real GPS/driver data is unavailable.
export const DEFAULT_DRIVER_ID = process.env.EXPO_PUBLIC_DEFAULT_DRIVER_ID || '';
export const DEFAULT_LATITUDE = Number(process.env.EXPO_PUBLIC_DEFAULT_LATITUDE || 18.5204);
export const DEFAULT_LONGITUDE = Number(process.env.EXPO_PUBLIC_DEFAULT_LONGITUDE || 73.8567);
export const DEFAULT_DESTINATION_LATITUDE = Number(process.env.EXPO_PUBLIC_DEFAULT_DESTINATION_LATITUDE || 18.5308);
export const DEFAULT_DESTINATION_LONGITUDE = Number(process.env.EXPO_PUBLIC_DEFAULT_DESTINATION_LONGITUDE || 73.8474);

export function getBackendHost(): string {
  return BACKEND_HOST;
}

export function getApiBaseURL(): string {
  return API_BASE_URL;
}

export function getGraphQLURL(): string {
  return `${API_BASE_URL}/query`;
}

export function getMQTTBrokerURL(): string {
  return MQTT_BROKER_URL;
}
