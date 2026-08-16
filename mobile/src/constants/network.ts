export const API_SCHEME = 'http';
export const MQTT_SCHEME = 'ws';
export const BACKEND_HOST = '127.0.0.1';
export const API_PORT = 8080;
export const MQTT_BROKER_PORT = 9001;

export const API_BASE_URL = `${API_SCHEME}://${BACKEND_HOST}:${API_PORT}`;
export const MQTT_BROKER_URL = `${MQTT_SCHEME}://${BACKEND_HOST}:${MQTT_BROKER_PORT}`;

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
