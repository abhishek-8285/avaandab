import Constants from 'expo-constants';

export function getBackendHost(): string {
  // Always use 127.0.0.1 since adb reverse forwards tcp:8080 & tcp:9001 directly over USB
  return '127.0.0.1';
}

export function getGraphQLURL(): string {
  const host = getBackendHost();
  return `http://${host}:8080/query`;
}

export function getMQTTBrokerURL(): string {
  const host = getBackendHost();
  return `ws://${host}:9001`;
}
