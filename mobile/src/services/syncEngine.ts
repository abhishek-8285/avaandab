import NetInfo from '@react-native-community/netinfo';
import { getApiBaseURL } from '../constants/network';
import { DB } from './storage';
import { OfflineQueue } from './offlineQueue';
import { useAuthStore } from '../stores/authStore';

let wasConnected = true;
let unsubscribeNetInfo: (() => void) | null = null;

export function startNetworkWatcher(): void {
  if (unsubscribeNetInfo) return;

  unsubscribeNetInfo = NetInfo.addEventListener((state) => {
    const isConnected = state.isConnected ?? false;

    if (isConnected && !wasConnected) {
      // Just reconnected — flush offline queues
      OfflineQueue.flush()
        .then(({ podsFlushed, gpsFlushed }) => {
          if (podsFlushed > 0 || gpsFlushed > 0) {
            console.log(`[OfflineQueue] Flushed ${podsFlushed} PODs, ${gpsFlushed} GPS logs on reconnect`);
          }
        })
        .catch((err) => {
          console.warn('[OfflineQueue] Flush failed:', err);
        });

      const driverId = useAuthStore.getState().user?.driverId || useAuthStore.getState().user?.id;
      if (driverId) {
        SyncEngine.syncPendingLogs(driverId).catch(() => {});
      }
    }

    wasConnected = isConnected;
  });
}

export function stopNetworkWatcher(): void {
  if (unsubscribeNetInfo) {
    unsubscribeNetInfo();
    unsubscribeNetInfo = null;
  }
}

class SyncEngineService {
  private syncTimer: NodeJS.Timeout | null = null;
  private isSyncing = false;

  startAutoSync(driverId: string, intervalMs = 15000): void {
    if (this.syncTimer) return;

    this.syncTimer = setInterval(() => {
      this.syncPendingLogs(driverId);
    }, intervalMs);

    console.log(`[SYNC ENGINE] Auto-sync background service started (${intervalMs / 1000}s interval)`);
  }

  stopAutoSync(): void {
    if (this.syncTimer) {
      clearInterval(this.syncTimer);
      this.syncTimer = null;
    }
  }

  async syncPendingLogs(driverId: string): Promise<{ syncedCount: number; error: string | null }> {
    if (this.isSyncing) return { syncedCount: 0, error: 'Sync already in progress' };

    this.isSyncing = true;
    try {
      const unsyncedLogs = await DB.getUnsyncedGPSLogs();
      if (!unsyncedLogs || unsyncedLogs.length === 0) {
        this.isSyncing = false;
        return { syncedCount: 0, error: null };
      }

      const syncEndpoint = `${getApiBaseURL()}/api/v1/telemetry/sync`;
      const token = useAuthStore.getState().token;

      const response = await fetch(syncEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          driver_id: driverId,
          logs: unsyncedLogs,
        }),
      });

      if (!response.ok) {
        throw new Error(`Server returned HTTP ${response.status}`);
      }

      const result = await response.json();
      if (result.success && Array.isArray(result.synced_ids)) {
        await DB.markLogsAsSynced(result.synced_ids);
        console.log(`[SYNC ENGINE SUCCESS] Synced & marked ${result.synced_ids.length} records in SQLite DB`);
        this.isSyncing = false;
        return { syncedCount: result.synced_ids.length, error: null };
      }

      this.isSyncing = false;
      return { syncedCount: 0, error: 'Unexpected server response' };
    } catch (err: any) {
      this.isSyncing = false;
      console.log('[SYNC ENGINE WARNING] Auto-sync failed (re-queueing for offline retention):', err.message);
      return { syncedCount: 0, error: err.message };
    }
  }
}

export const SyncEngine = new SyncEngineService();
