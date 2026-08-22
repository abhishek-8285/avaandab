import { getApiBaseURL, getBackendHost, getMQTTBrokerURL } from '../src/constants/network';

describe('network constants', () => {
  test('returns default baseURL and host', () => {
    expect(getBackendHost()).toBeDefined();
    expect(getApiBaseURL()).toBeDefined();
    expect(getMQTTBrokerURL()).toBeDefined();
  });
});

describe('network constants (env overrides)', () => {
  const ORIGINAL_DEV = (global as any).__DEV__;

  afterEach(() => {
    (global as any).__DEV__ = ORIGINAL_DEV;
  });

  test('production defaults use https/wss and standard ports', () => {
    (global as any).__DEV__ = false;
    let net: any;
    jest.isolateModules(() => {
      net = require('../src/constants/network');
    });
    expect(net.API_SCHEME).toBe('https');
    expect(net.MQTT_SCHEME).toBe('wss');
    expect(net.getBackendHost()).toBe('api.avandab.com');
    expect(net.getApiBaseURL()).toBe('https://api.avandab.com');
    expect(net.getMQTTBrokerURL()).toBe('wss://api.avandab.com:8883');
  });

  test('dev defaults use http/ws on localhost with explicit ports', () => {
    (global as any).__DEV__ = true;
    let net: any;
    jest.isolateModules(() => {
      net = require('../src/constants/network');
    });
    expect(net.API_SCHEME).toBe('http');
    expect(net.MQTT_SCHEME).toBe('ws');
    expect(net.getBackendHost()).toBe('127.0.0.1');
    expect(net.getApiBaseURL()).toBe('http://127.0.0.1:8080');
    expect(net.getMQTTBrokerURL()).toBe('ws://127.0.0.1:9001');
  });
});
