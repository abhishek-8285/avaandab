import { OfflineQueue } from '../src/services/offlineQueue';
import { resetSQLiteMockState } from '../jest/setup';
import { useAuthStore } from '../src/stores/authStore';

// Mock fetch globally
const globalFetch = global.fetch;

describe('OfflineQueue', () => {
  beforeEach(async () => {
    resetSQLiteMockState();
    await OfflineQueue.init();
    await useAuthStore.getState().setAuth('mock_token_123', {
      id: 'u_1',
      name: 'Rajesh Kumar',
      role: 'driver',
      email: 'driver@avandab.com',
      driverId: 'drv_1',
    });
  });

  afterEach(() => {
    global.fetch = globalFetch;
  });

  test('enqueuePOD stores in SQLite and pendingPODs returns it', async () => {
    await OfflineQueue.enqueuePOD('trip_101', {
      consignee_name: 'Acme Logistics',
      notes: 'Received at Dock 4',
      photo_uri: 'file:///photo.jpg',
      latitude: 18.5204,
      longitude: 73.8567,
    });

    const pods = await OfflineQueue.pendingPODs();
    expect(pods).toHaveLength(1);
    expect(pods[0].trip_id).toBe('trip_101');
    expect(pods[0].consignee_name).toBe('Acme Logistics');
    expect(pods[0].notes).toBe('Received at Dock 4');
  });

  test('enqueuePOD dedupes by tripId (no duplicate rows)', async () => {
    await OfflineQueue.enqueuePOD('trip_102', { consignee_name: 'Buyer A' });
    await OfflineQueue.enqueuePOD('trip_102', { consignee_name: 'Buyer A duplicate' });

    const pods = await OfflineQueue.pendingPODs();
    expect(pods.filter((p) => p.trip_id === 'trip_102')).toHaveLength(1);
  });

  test('clearPOD removes the row', async () => {
    await OfflineQueue.enqueuePOD('trip_103', { consignee_name: 'Buyer B' });
    await OfflineQueue.clearPOD('trip_103');

    const pods = await OfflineQueue.pendingPODs();
    expect(pods.find((p) => p.trip_id === 'trip_103')).toBeUndefined();
  });

  test('clearGPS with empty array does nothing', async () => {
    await expect(OfflineQueue.clearGPS([])).resolves.not.toThrow();
  });

  test('enqueueGPS stores, pendingGPS returns, clearGPS removes', async () => {
    await OfflineQueue.enqueueGPS({
      driver_id: 'drv_1',
      latitude: 18.5204,
      longitude: 73.8567,
      timestamp: new Date().toISOString(),
      accuracy_m: 5.0,
    });

    const gpsLogs = await OfflineQueue.pendingGPS();
    expect(gpsLogs).toHaveLength(1);
    expect(gpsLogs[0].driver_id).toBe('drv_1');

    await OfflineQueue.clearGPS([gpsLogs[0].id]);
    const afterClear = await OfflineQueue.pendingGPS();
    expect(afterClear).toHaveLength(0);
  });

  test('flush with network succeeds and clears flushed items with photos and coordinates', async () => {
    await OfflineQueue.enqueuePOD('trip_flush_1', {
      consignee_name: 'Receiver 1',
      notes: 'Dock notes',
      photo_uri: 'file:///photo.jpg',
      latitude: 18.5204,
      longitude: 73.8567,
    });
    await OfflineQueue.enqueueGPS({
      driver_id: 'drv_1',
      latitude: 18.5204,
      longitude: 73.8567,
      timestamp: new Date().toISOString(),
    });

    global.fetch = jest.fn().mockImplementation(async (url: string) => {
      if (url.includes('/deliver-pod')) {
        return { ok: true, json: async () => ({ status: 'delivered', trip_number: 'TRP-1' }) };
      }
      if (url.includes('/telemetry/sync')) {
        return { ok: true, json: async () => ({ success: true, synced_ids: [1] }) };
      }
      return { ok: true, json: async () => ({}) };
    }) as any;

    const result = await OfflineQueue.flush();
    expect(result.podsFlushed).toBe(1);
    expect(result.gpsFlushed).toBe(1);

    const remainingPods = await OfflineQueue.pendingPODs();
    expect(remainingPods).toHaveLength(0);
  });

  test('flush without token stops early', async () => {
    await useAuthStore.getState().logout();
    await OfflineQueue.enqueuePOD('trip_unauth', { consignee_name: 'Unauth' });
    await OfflineQueue.enqueueGPS({ driver_id: 'd1', latitude: 1, longitude: 2, timestamp: '' });

    const result = await OfflineQueue.flush();
    expect(result.podsFlushed).toBe(0);
    expect(result.gpsFlushed).toBe(0);
  });

  test('flush without network leaves items in queue', async () => {
    await OfflineQueue.enqueuePOD('trip_flush_offline', { consignee_name: 'Receiver Offline' });
    await OfflineQueue.enqueueGPS({ driver_id: 'd1', latitude: 1, longitude: 2, timestamp: '' });

    global.fetch = jest.fn().mockRejectedValue(new Error('Network error')) as any;

    const result = await OfflineQueue.flush();
    expect(result.podsFlushed).toBe(0);
    expect(result.gpsFlushed).toBe(0);

    const remaining = await OfflineQueue.pendingPODs();
    expect(remaining).toHaveLength(1);
  });
});
