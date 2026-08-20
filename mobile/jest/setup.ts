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
  getItemAsync: jest.fn(),
  setItemAsync: jest.fn(),
  deleteItemAsync: jest.fn(),
}));

// Mock expo-location (native module)
jest.mock('expo-location', () => ({
  requestForegroundPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted' }),
  watchPositionAsync: jest.fn(),
  getCurrentPositionAsync: jest.fn(),
}));

// Mock expo-camera (native module)
jest.mock('expo-camera', () => ({
  Camera: jest.fn(),
  CameraView: jest.fn(),
  useCameraPermissions: jest.fn().mockReturnValue([
    { granted: true, status: 'granted' },
    jest.fn().mockResolvedValue({ granted: true, status: 'granted' }),
  ]),
  requestCameraPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted', granted: true }),
}));

// In-memory mock database state for expo-sqlite
const sqliteMockState = {
  queued_pods: [] as any[],
  queued_gps: [] as any[],
  trips: [] as any[],
  offline_gps_logs: [] as any[],
};

export const resetSQLiteMockState = () => {
  sqliteMockState.queued_pods = [];
  sqliteMockState.queued_gps = [];
  sqliteMockState.trips = [];
  sqliteMockState.offline_gps_logs = [];
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
      return [];
    }),
    runAsync: jest.fn().mockImplementation(async (query: string, params: any[] = []) => {
      if (query.includes('INSERT INTO queued_pods')) {
        const pod = {
          id: sqliteMockState.queued_pods.length + 1,
          trip_id: params[0],
          consignee_name: params[1],
          notes: params[2],
          photo_uri: params[3],
          latitude: params[4],
          longitude: params[5],
          created_at: new Date().toISOString(),
        };
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
