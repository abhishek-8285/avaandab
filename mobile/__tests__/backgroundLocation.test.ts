import { resetSQLiteMockState, getSQLiteMockState } from '../jest/setup';
import { useAuthStore } from '../src/stores/authStore';

jest.mock('expo-task-manager', () => ({
  __esModule: true,
  default: { defineTask: jest.fn() },
  defineTask: jest.fn(),
}));

jest.mock('expo-location', () => ({
  requestForegroundPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted', granted: true }),
  getForegroundPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted', granted: true }),
  requestBackgroundPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted', granted: true }),
  hasServicesEnabledAsync: jest.fn().mockResolvedValue(true),
  getLastKnownPositionAsync: jest.fn().mockResolvedValue(null),
  getCurrentPositionAsync: jest.fn().mockResolvedValue({ coords: { latitude: 19.076, longitude: 72.8777 } }),
  watchPositionAsync: jest.fn(),
  startLocationUpdatesAsync: jest.fn().mockResolvedValue(undefined),
  stopLocationUpdatesAsync: jest.fn().mockResolvedValue(undefined),
  hasStartedLocationUpdatesAsync: jest.fn().mockResolvedValue(false),
  Accuracy: { Balanced: 3, High: 4, Low: 1 },
}));

jest.mock('../src/services/mqtt', () => ({
  MQTT: { publishLocation: jest.fn() },
}));

/**
 * setup.ts clears all mocks in beforeEach, wiping the module-load-time
 * defineTask call. isolateModules gives each test its own module sandbox so
 * every assertion sees the exact instances the service used.
 */
function loadFreshService() {
  let Loc: any, MQ: any, svc: any, Store: any;
  jest.isolateModules(() => {
    Loc = require('expo-location');
    MQ = require('../src/services/mqtt');
    Store = require('../src/stores/authStore');
    svc = require('../src/services/backgroundLocation');
  });
  return { Loc, svc, MQ: MQ.MQTT, Store: Store.useAuthStore };
}

describe('BackgroundGPS', () => {
  beforeEach(async () => {
    resetSQLiteMockState();
    await useAuthStore.getState().setAuth('tok', {
      id: 'u_1', name: 'Raj', role: 'driver', email: 'r@x.com', driverId: 'drv_9',
    });
  });

  test('task handler is exported and wired under the fixed task name', () => {
    const { svc } = loadFreshService();
    expect(svc.BACKGROUND_LOCATION_TASK).toBe('AVANDAB_BACKGROUND_GPS');
    expect(typeof svc.backgroundGPSTask).toBe('function');
    // The exact function handed to TaskManager.defineTask is what we export.
    expect(svc.BackgroundGPS.taskHandler).toBe(svc.backgroundGPSTask);
  });

  test('start requests permissions and starts OS updates', async () => {
    const { Loc, svc } = loadFreshService();
    const res = await svc.BackgroundGPS.start();
    expect(res.started).toBe(true);
    expect(res.error).toBeNull();
    expect(Loc.requestForegroundPermissionsAsync).toHaveBeenCalled();
    expect(Loc.requestBackgroundPermissionsAsync).toHaveBeenCalled();
    expect(Loc.startLocationUpdatesAsync).toHaveBeenCalledWith(
      'AVANDAB_BACKGROUND_GPS',
      expect.objectContaining({ timeInterval: 15000 })
    );
  });

  test('start fails honestly when background permission denied', async () => {
    const { Loc, svc } = loadFreshService();
    (Loc.requestBackgroundPermissionsAsync as jest.Mock).mockResolvedValueOnce({ status: 'denied' });
    const res = await svc.BackgroundGPS.start();
    expect(res.started).toBe(false);
    expect(res.error).toContain('Background location permission denied');
    expect(Loc.startLocationUpdatesAsync).not.toHaveBeenCalled();
  });

  test('stop is safe when task not running', async () => {
    const { svc } = loadFreshService();
    await expect(svc.BackgroundGPS.stop()).resolves.not.toThrow();
  });

  test('isRunning reports OS task state', async () => {
    const { Loc, svc } = loadFreshService();
    (Loc.hasStartedLocationUpdatesAsync as jest.Mock).mockResolvedValueOnce(true);
    await expect(svc.BackgroundGPS.isRunning()).resolves.toBe(true);
  });

  test('task handler persists fixes, publishes MQTT, echoes to UI', async () => {
    const { svc, MQ, Store } = loadFreshService();
    // The sandboxed module graph has its own store instance — auth it here.
    await Store.getState().setAuth('tok', {
      id: 'u_1', name: 'Raj', role: 'driver', email: 'r@x.com', driverId: 'drv_9',
    });
    const handler = svc.backgroundGPSTask as (evt: any) => Promise<void>;

    const echo = jest.fn();
    svc.BackgroundGPS.setForegroundEcho(echo);

    await handler({
      data: {
        locations: [
          { coords: { latitude: 28.71, longitude: 77.1, accuracy: 7.5, speed: 5.5 } },
          { coords: {} }, // malformed — skipped silently
        ],
      },
      error: undefined,
    });

    const logs = getSQLiteMockState().offline_gps_logs;
    expect(logs).toHaveLength(1);
    expect(logs[0].latitude).toBe(28.71);
    expect(logs[0].accuracy).toBe(7.5);
    expect(MQ.publishLocation).toHaveBeenCalledWith('drv_9', 28.71, 77.1);
    expect(echo).toHaveBeenCalledWith(28.71, 77.1, 20); // 5.5 m/s ≈ 20 km/h
  });

  test('task handler tolerates errors without throwing', async () => {
    const { svc } = loadFreshService();
    await expect((svc.backgroundGPSTask as (evt: any) => Promise<void>)({ data: {}, error: new Error('gps lost') })).resolves.toBeUndefined();
  });
});
