import * as SQLite from 'expo-sqlite';
import { getApiBaseURL } from '../constants/network';
import { useAuthStore } from '../stores/authStore';

const DB_NAME = 'offline_queue.db';

export interface QueuedPOD {
  id: number;
  trip_id: string;
  consignee_name: string;
  notes: string;
  photo_uri: string | null;
  latitude: number | null;
  longitude: number | null;
  created_at: string;
}

export interface QueuedGPS {
  id: number;
  driver_id: string;
  latitude: number;
  longitude: number;
  timestamp: string;
  accuracy_m: number | null;
  created_at: string;
}

class OfflineQueueService {
  private db: SQLite.SQLiteDatabase | null = null;

  async init(): Promise<void> {
    if (this.db) return;
    this.db = await SQLite.openDatabaseAsync(DB_NAME);
    await this.db.execAsync(`
      CREATE TABLE IF NOT EXISTS queued_pods (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        trip_id TEXT NOT NULL,
        consignee_name TEXT NOT NULL DEFAULT '',
        notes TEXT NOT NULL DEFAULT '',
        photo_uri TEXT,
        latitude REAL,
        longitude REAL,
        created_at TEXT NOT NULL DEFAULT (datetime('now'))
      );
      CREATE TABLE IF NOT EXISTS queued_gps (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        driver_id TEXT NOT NULL,
        latitude REAL NOT NULL,
        longitude REAL NOT NULL,
        timestamp TEXT NOT NULL,
        accuracy_m REAL,
        created_at TEXT NOT NULL DEFAULT (datetime('now'))
      );
    `);
  }

  // ── POD queue ──────────────────────────────────────────
  async enqueuePOD(
    tripId: string,
    data: {
      consignee_name: string;
      notes?: string;
      photo_uri?: string | null;
      latitude?: number | null;
      longitude?: number | null;
    }
  ): Promise<void> {
    if (!this.db) await this.init();
    // Dedupe: don't queue twice for the same trip
    const existing = await this.db!.getFirstAsync<QueuedPOD>(
      'SELECT id FROM queued_pods WHERE trip_id = ?',
      [tripId]
    );
    if (existing) return;

    await this.db!.runAsync(
      `INSERT INTO queued_pods (trip_id, consignee_name, notes, photo_uri, latitude, longitude)
       VALUES (?, ?, ?, ?, ?, ?)`,
      [
        tripId,
        data.consignee_name,
        data.notes || '',
        data.photo_uri || null,
        data.latitude ?? null,
        data.longitude ?? null,
      ]
    );
  }

  async clearPOD(tripId: string): Promise<void> {
    if (!this.db) await this.init();
    await this.db!.runAsync('DELETE FROM queued_pods WHERE trip_id = ?', [tripId]);
  }

  async pendingPODs(): Promise<QueuedPOD[]> {
    if (!this.db) await this.init();
    return await this.db!.getAllAsync<QueuedPOD>('SELECT * FROM queued_pods ORDER BY created_at ASC');
  }

  // ── GPS queue ──────────────────────────────────────────
  async enqueueGPS(log: {
    driver_id: string;
    latitude: number;
    longitude: number;
    timestamp: string;
    accuracy_m?: number | null;
  }): Promise<void> {
    if (!this.db) await this.init();
    await this.db!.runAsync(
      `INSERT INTO queued_gps (driver_id, latitude, longitude, timestamp, accuracy_m)
       VALUES (?, ?, ?, ?, ?)`,
      [log.driver_id, log.latitude, log.longitude, log.timestamp, log.accuracy_m ?? null]
    );
  }

  async pendingGPS(): Promise<QueuedGPS[]> {
    if (!this.db) await this.init();
    return await this.db!.getAllAsync<QueuedGPS>('SELECT * FROM queued_gps ORDER BY created_at ASC');
  }

  async clearGPS(ids: number[]): Promise<void> {
    if (!this.db) await this.init();
    if (ids.length === 0) return;
    const placeholders = ids.map(() => '?').join(',');
    await this.db!.runAsync(`DELETE FROM queued_gps WHERE id IN (${placeholders})`, ids);
  }

  // ── Flush all queues ───────────────────────────────────
  async flush(): Promise<{ podsFlushed: number; gpsFlushed: number }> {
    let podsFlushed = 0;
    let gpsFlushed = 0;

    // Flush queued PODs
    const pods = await this.pendingPODs();
    for (const pod of pods) {
      try {
        const token = useAuthStore.getState().token;
        if (!token) break;

        const form = new FormData();
        form.append('consignee_name', pod.consignee_name);
        form.append('notes', pod.notes);
        if (pod.photo_uri) {
          form.append('pod_photo', {
            uri: pod.photo_uri,
            name: 'pod.jpg',
            type: 'image/jpeg',
          } as any);
        }
        if (pod.latitude != null && pod.longitude != null) {
          form.append('latitude', String(pod.latitude));
          form.append('longitude', String(pod.longitude));
        }

        const res = await fetch(`${getApiBaseURL()}/api/v1/trips/${pod.trip_id}/deliver-pod`, {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}` },
          body: form,
        });

        if (res.ok) {
          await this.clearPOD(pod.trip_id);
          podsFlushed++;
        }
      } catch {
        break; // Network still down, stop flushing
      }
    }

    // Flush queued GPS
    const gpsLogs = await this.pendingGPS();
    if (gpsLogs.length > 0) {
      try {
        const token = useAuthStore.getState().token;
        const driverId = useAuthStore.getState().user?.driverId || useAuthStore.getState().user?.id;
        if (token && driverId) {
          const res = await fetch(`${getApiBaseURL()}/api/v1/telemetry/sync`, {
            method: 'POST',
            headers: {
              Authorization: `Bearer ${token}`,
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({
              driver_id: driverId,
              logs: gpsLogs.map((g) => ({
                latitude: g.latitude,
                longitude: g.longitude,
                timestamp: g.timestamp,
                accuracy_m: g.accuracy_m,
              })),
            }),
          });
          if (res.ok) {
            const json = await res.json();
            if (json.success) {
              await this.clearGPS(gpsLogs.map((g) => g.id));
              gpsFlushed = gpsLogs.length;
            }
          }
        }
      } catch {
        // Network still down
      }
    }

    return { podsFlushed, gpsFlushed };
  }
}

export const OfflineQueue = new OfflineQueueService();
