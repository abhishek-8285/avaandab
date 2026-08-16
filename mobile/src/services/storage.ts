import AsyncStorage from '@react-native-async-storage/async-storage';
import * as SQLite from 'expo-sqlite';
import { Trip, Driver } from '../types/api';

const KEYS = {
  OFFLINE_TRIPS: '@avandab_offline_trips',
};

// ==========================================
// 1. Key-Value Storage (User Prefs)
// ==========================================
export const Storage = {
  async saveOfflineTrips(trips: Trip[]): Promise<void> {
    await AsyncStorage.setItem(KEYS.OFFLINE_TRIPS, JSON.stringify(trips));
  },

  async getOfflineTrips(): Promise<Trip[]> {
    const json = await AsyncStorage.getItem(KEYS.OFFLINE_TRIPS);
    return json ? JSON.parse(json) : [];
  },
};

// ==========================================
// 2. High-Performance SQLite (Structured Offline Data)
// ==========================================
let db: SQLite.SQLiteDatabase | null = null;

export const initDatabase = async (): Promise<void> => {
  if (db) return;
  db = await SQLite.openDatabaseAsync('avandab_offline.db');

  await db.execAsync(`
    PRAGMA journal_mode = WAL;
    CREATE TABLE IF NOT EXISTS trips (
      id TEXT PRIMARY KEY NOT NULL,
      tripNumber TEXT NOT NULL,
      driverName TEXT NOT NULL,
      vehiclePlate TEXT NOT NULL,
      origin TEXT NOT NULL,
      destination TEXT NOT NULL,
      status TEXT NOT NULL,
      startTime TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS offline_gps_logs (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      latitude REAL NOT NULL,
      longitude REAL NOT NULL,
      timestamp TEXT NOT NULL,
      synced INTEGER DEFAULT 0
    );
  `);
};

export const DB = {
  async saveTrips(trips: Trip[]): Promise<void> {
    await initDatabase();
    if (!db) return;

    for (const trip of trips) {
      await db.runAsync(
        `INSERT OR REPLACE INTO trips (id, tripNumber, driverName, vehiclePlate, origin, destination, status, startTime)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?);`,
        [trip.id, trip.tripNumber, trip.driverName, trip.vehiclePlate, trip.origin, trip.destination, trip.status, trip.startTime]
      );
    }
  },

  async getTrips(): Promise<Trip[]> {
    await initDatabase();
    if (!db) return [];

    const rows = await db.getAllAsync<Trip>('SELECT * FROM trips ORDER BY startTime DESC;');
    return rows;
  },

  async logGPSLocation(lat: number, lng: number): Promise<void> {
    await initDatabase();
    if (!db) return;

    await db.runAsync(
      'INSERT INTO offline_gps_logs (latitude, longitude, timestamp) VALUES (?, ?, ?);',
      [lat, lng, new Date().toISOString()]
    );
  },

  async getUnsyncedGPSLogs(): Promise<Array<{ id: number; latitude: number; longitude: number; timestamp: string }>> {
    await initDatabase();
    if (!db) return [];

    const rows = await db.getAllAsync<{ id: number; latitude: number; longitude: number; timestamp: string }>(
      `SELECT id, latitude, longitude, timestamp FROM offline_gps_logs WHERE synced = 0 ORDER BY id ASC LIMIT 50;`
    );
    return rows;
  },

  async markLogsAsSynced(ids: number[]): Promise<void> {
    if (!ids || ids.length === 0) return;
    await initDatabase();
    if (!db) return;

    const placeholders = ids.map(() => '?').join(',');
    await db.runAsync(
      `UPDATE offline_gps_logs SET synced = 1 WHERE id IN (${placeholders});`,
      ids
    );
  },
};
