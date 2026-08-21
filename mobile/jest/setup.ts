// Reset mocks between tests to prevent cross-test contamination.
import '@testing-library/react-native/extend-expect';

beforeEach(() => {
  jest.clearAllMocks();
});

afterEach(() => {
  jest.restoreAllMocks();
});

// Mock @expo/vector-icons and expo-font (native modules)
jest.mock('@expo/vector-icons', () => ({
  MaterialCommunityIcons: 'MaterialCommunityIcons',
  Ionicons: 'Ionicons',
  FontAwesome: 'FontAwesome',
}));

jest.mock('expo-font', () => ({
  isLoaded: jest.fn().mockReturnValue(true),
  loadAsync: jest.fn().mockResolvedValue(undefined),
}));
jest.mock('expo-secure-store', () => ({
  getItemAsync: jest.fn().mockResolvedValue(null),
  setItemAsync: jest.fn().mockResolvedValue(undefined),
  deleteItemAsync: jest.fn().mockResolvedValue(undefined),
}));

// Mock expo-location (native module)
jest.mock('expo-location', () => ({
  requestForegroundPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted' }),
  getForegroundPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted' }),
  hasServicesEnabledAsync: jest.fn().mockResolvedValue(true),
  getLastKnownPositionAsync: jest.fn().mockResolvedValue(null),
  getCurrentPositionAsync: jest.fn().mockResolvedValue({ coords: { latitude: 19.076, longitude: 72.8777 } }),
  watchPositionAsync: jest.fn(),
  Accuracy: { Balanced: 3, High: 4, Low: 1 },
}));

// Mock expo-camera (native module)
jest.mock('expo-camera', () => ({
  Camera: { requestCameraPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted', granted: true }) },
  CameraView: 'CameraView',
  useCameraPermissions: jest.fn().mockReturnValue([
    { granted: true, status: 'granted' },
    jest.fn().mockResolvedValue({ granted: true, status: 'granted' }),
  ]),
  requestCameraPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted', granted: true }),
}));

// Mock expo-image-picker
jest.mock('expo-image-picker', () => ({
  requestMediaLibraryPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted', granted: true }),
  requestCameraPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted', granted: true }),
  launchImageLibraryAsync: jest.fn().mockResolvedValue({ canceled: true, assets: [] }),
  launchCameraAsync: jest.fn().mockResolvedValue({ canceled: true, assets: [] }),
  MediaTypeOptions: { Images: 'Images', Videos: 'Videos', All: 'All' },
}), { virtual: true });

// Mock react-native-signature-canvas
jest.mock('react-native-signature-canvas', () => 'SignaturePad', { virtual: true });
jest.mock('react-native-webview', () => ({
  WebView: 'WebView',
}), { virtual: true });

// Mock i18next / react-i18next
jest.mock('i18next', () => ({
  use: jest.fn().mockReturnThis(),
  init: jest.fn().mockResolvedValue(undefined),
  t: jest.fn((k: string) => k),
  changeLanguage: jest.fn().mockResolvedValue(undefined),
  language: 'en',
}), { virtual: true });
jest.mock('react-i18next', () => ({
  initReactI18next: { type: '3rdParty', init: jest.fn() },
  useTranslation: () => ({ t: (k: string) => k, i18n: { language: 'en', changeLanguage: jest.fn() } }),
}), { virtual: true });

// Mock @react-native-async-storage/async-storage
jest.mock('@react-native-async-storage/async-storage', () => ({
  default: {
    getItem: jest.fn().mockResolvedValue(null),
    setItem: jest.fn().mockResolvedValue(undefined),
    removeItem: jest.fn().mockResolvedValue(undefined),
  },
  getItem: jest.fn().mockResolvedValue(null),
  setItem: jest.fn().mockResolvedValue(undefined),
  removeItem: jest.fn().mockResolvedValue(undefined),
}));

// In-memory mock database state for expo-sqlite
const sqliteMockState = {
  queued_pods: [] as any[],
  queued_gps: [] as any[],
  trips: [] as any[],
  offline_gps_logs: [] as any[],
  offline_expenses: [] as any[],
};

export const getSQLiteMockState = () => sqliteMockState;

export const resetSQLiteMockState = () => {
  sqliteMockState.queued_pods = [];
  sqliteMockState.queued_gps = [];
  sqliteMockState.trips = [];
  sqliteMockState.offline_gps_logs = [];
  sqliteMockState.offline_expenses = [];
};

// Mock expo-sqlite (native module)
jest.mock('expo-sqlite', () => ({
  openDatabaseAsync: jest.fn().mockResolvedValue({
    execAsync: jest.fn().mockResolvedValue(undefined),
    getFirstAsync: jest.fn().mockImplementation(async (query: string, params: any[] = []) => {
      if (query.includes('queued_pods WHERE trip_id =')) {
        return sqliteMockState.queued_pods.find((p) => p.trip_id === params[0]) || null;
      }
      return null;
    }),
    getAllAsync: jest.fn().mockImplementation(async (query: string) => {
      if (query.includes('queued_pods')) {
        return [...sqliteMockState.queued_pods];
      }
      if (query.includes('queued_gps')) {
        return [...sqliteMockState.queued_gps];
      }
      if (query.includes('trips')) {
        return [...sqliteMockState.trips];
      }
      if (query.includes('offline_gps_logs')) {
        return [...sqliteMockState.offline_gps_logs];
      }
      if (query.includes('offline_expenses')) {
        return [...sqliteMockState.offline_expenses];
      }
      return [];
    }),
    runAsync: jest.fn().mockImplementation(async (query: string, params: any[] = []) => {
      if (query.includes('INSERT INTO queued_pods')) {
        // New schema: (trip_id, consignee_name, consignee_phone, notes, photo_uri, latitude, longitude, pod_signature_data, quantity_short, damage_qty, refusal_reason)
        // Old schema fallback: (trip_id, consignee_name, notes, photo_uri, latitude, longitude)
        let pod: any;
        if (query.includes('consignee_phone')) {
          pod = {
            id: sqliteMockState.queued_pods.length + 1,
            trip_id: params[0],
            consignee_name: params[1],
            consignee_phone: params[2] ?? null,
            notes: params[3],
            photo_uri: params[4],
            latitude: params[5],
            longitude: params[6],
            pod_signature_data: params[7] ?? null,
            quantity_short: params[8] ?? null,
            damage_qty: params[9] ?? null,
            refusal_reason: params[10] ?? null,
            created_at: new Date().toISOString(),
          };
        } else {
          // fallback old
          pod = {
            id: sqliteMockState.queued_pods.length + 1,
            trip_id: params[0],
            consignee_name: params[1],
            consignee_phone: null,
            notes: params[2],
            photo_uri: params[3],
            latitude: params[4],
            longitude: params[5],
            pod_signature_data: null,
            quantity_short: null,
            damage_qty: null,
            refusal_reason: null,
            created_at: new Date().toISOString(),
          };
        }
        sqliteMockState.queued_pods.push(pod);
      } else if (query.includes('DELETE FROM queued_pods WHERE trip_id =')) {
        sqliteMockState.queued_pods = sqliteMockState.queued_pods.filter((p) => p.trip_id !== params[0]);
      } else if (query.includes('INSERT INTO queued_gps')) {
        const gps = {
          id: sqliteMockState.queued_gps.length + 1,
          driver_id: params[0],
          latitude: params[1],
          longitude: params[2],
          timestamp: params[3],
          accuracy_m: params[4],
          created_at: new Date().toISOString(),
        };
        sqliteMockState.queued_gps.push(gps);
      } else if (query.includes('DELETE FROM queued_gps WHERE id IN')) {
        const ids = params;
        sqliteMockState.queued_gps = sqliteMockState.queued_gps.filter((g) => !ids.includes(g.id));
      } else if (query.includes('INSERT INTO offline_expenses')) {
        const exp = {
          id: sqliteMockState.offline_expenses.length + 1,
          trip_id: params[0],
          expense_type: params[1],
          amount: params[2],
          receipt_uri: params[3],
          notes: params[4],
          latitude: params[5],
          longitude: params[6],
          created_at: new Date().toISOString(),
        };
        sqliteMockState.offline_expenses.push(exp);
      } else if (query.includes('DELETE FROM offline_expenses WHERE id =')) {
        sqliteMockState.offline_expenses = sqliteMockState.offline_expenses.filter((e) => e.id !== params[0]);
      } else if (query.includes('DELETE FROM offline_expenses WHERE id IN')) {
        const ids = params;
        sqliteMockState.offline_expenses = sqliteMockState.offline_expenses.filter((e) => !ids.includes(e.id));
      } else if (query.includes('INSERT OR REPLACE INTO trips') || query.includes('INSERT INTO trips')) {
        const trip = {
          id: params[0],
          tripNumber: params[1],
          driverName: params[2],
          vehiclePlate: params[3],
          origin: params[4],
          destination: params[5],
          status: params[6],
          startTime: params[7],
        };
        const idx = sqliteMockState.trips.findIndex((t) => t.id === trip.id);
        if (idx >= 0) sqliteMockState.trips[idx] = trip;
        else sqliteMockState.trips.push(trip);
      } else if (query.includes('INSERT INTO offline_gps_logs')) {
        const log = {
          id: sqliteMockState.offline_gps_logs.length + 1,
          latitude: params[0],
          longitude: params[1],
          timestamp: params[2],
          synced: 0,
        };
        sqliteMockState.offline_gps_logs.push(log);
      } else if (query.includes('UPDATE offline_gps_logs SET synced = 1')) {
        const ids = params;
        sqliteMockState.offline_gps_logs.forEach((l) => {
          if (ids.includes(l.id)) l.synced = 1;
        });
      }
      return { lastInsertRowId: 1, changes: 1 };
    }),
  }),
}));

// Mock expo-image-manipulator (native module)
jest.mock('expo-image-manipulator', () => ({
  SaveFormat: { JPEG: 'jpeg', PNG: 'png', WEBP: 'webp' },
  manipulateAsync: jest.fn().mockImplementation(async (uri: string) => ({
    uri: `${uri}_compressed.jpg`,
    width: 800,
    height: 600,
  })),
}));

// Mock @react-native-community/netinfo
jest.mock('@react-native-community/netinfo', () => ({
  addEventListener: jest.fn(),
  fetch: jest.fn().mockResolvedValue({ isConnected: true }),
}));

// Mock mqtt (network module)
jest.mock('mqtt', () => ({
  connect: jest.fn().mockReturnValue({
    on: jest.fn(),
    publish: jest.fn(),
    subscribe: jest.fn(),
    end: jest.fn(),
  }),
}));
