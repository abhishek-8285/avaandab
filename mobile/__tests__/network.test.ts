import { getApiBaseURL, getBackendHost, getMQTTBrokerURL } from '../src/constants/network';

describe('network constants', () => {
  test('returns default baseURL and host', () => {
    expect(getBackendHost()).toBeDefined();
    expect(getApiBaseURL()).toBeDefined();
    expect(getMQTTBrokerURL()).toBeDefined();
  });
});
