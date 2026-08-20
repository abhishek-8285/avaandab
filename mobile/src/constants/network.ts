const isLocalDev = typeof __DEV__ !== 'undefined' ? Boolean(__DEV__) : false;
export const API_SCHEME = process.env.EXPO_PUBLIC_API_SCHEME || (isLocalDev ? 'http' : 'https');
export const MQTT_SCHEME = process.env.EXPO_PUBLIC_MQTT_SCHEME || (isLocalDev ? 'ws' : 'wss');
export const BACKEND_HOST = process.env.EXPO_PUBLIC_BACKEND_HOST || (isLocalDev ? '127.0.0.1' : 'api.avandab.com');
export const API_PORT = process.env.EXPO_PUBLIC_API_PORT ? Number(process.env.EXPO_PUBLIC_API_PORT) : (isLocalDev ? 8080 : 443);
export const MQTT_BROKER_PORT = process.env.EXPO_PUBLIC_MQTT_BROKER_PORT ? Number(process.env.EXPO_PUBLIC_MQTT_BROKER_PORT) : (isLocalDev ? 9001 : 8883);

export const API_BASE_URL = API_PORT === 443 || API_PORT === 80
  ? `${API_SCHEME}://${BACKEND_HOST}`
  : `${API_SCHEME}://${BACKEND_HOST}:${API_PORT}`;
export const MQTT_BROKER_URL = MQTT_BROKER_PORT === 443 || MQTT_BROKER_PORT === 80
  ? `${MQTT_SCHEME}://${BACKEND_HOST}`
  : `${MQTT_SCHEME}://${BACKEND_HOST}:${MQTT_BROKER_PORT}`;

export function getBackendHost(): string {
  return BACKEND_HOST;
}

export function getApiBaseURL(): string {
  return API_BASE_URL;
}

export function getMQTTBrokerURL(): string {
  return MQTT_BROKER_URL;
}
