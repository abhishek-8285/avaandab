import { DB } from '../src/services/storage';
import { getSQLiteMockState, resetSQLiteMockState } from '../jest/setup';

describe('DB GPS log storage (accuracy support)', () => {
  beforeEach(() => {
    resetSQLiteMockState();
  });

  test('logGPSLocation persists accuracy alongside coordinates', async () => {
    await DB.logGPSLocation(18.5204, 73.8567, 12.5);

    const state = getSQLiteMockState();
    expect(state.offline_gps_logs).toHaveLength(1);
    expect(state.offline_gps_logs[0].latitude).toBe(18.5204);
    expect(state.offline_gps_logs[0].longitude).toBe(73.8567);
    expect(state.offline_gps_logs[0].accuracy).toBe(12.5);
  });

  test('logGPSLocation tolerates missing accuracy (stored as null)', async () => {
    await DB.logGPSLocation(19.076, 72.8777);

    const state = getSQLiteMockState();
    expect(state.offline_gps_logs[0].accuracy).toBeNull();
  });

  test('getUnsyncedGPSLogs exposes accuracy_m and filters synced rows', async () => {
    await DB.logGPSLocation(18.5204, 73.8567, 8.0);

    let logs = await DB.getUnsyncedGPSLogs();
    expect(logs).toHaveLength(1);
    expect(logs[0].accuracy_m).toBe(8.0);

    await DB.markLogsAsSynced([logs[0].id]);
    logs = await DB.getUnsyncedGPSLogs();
    expect(logs).toHaveLength(0);
  });
});
