import * as SecureStore from 'expo-secure-store';
import { useAuthStore } from '../src/stores/authStore';

describe('authStore', () => {
  beforeEach(async () => {
    await useAuthStore.getState().logout();
  });

  test('setAuth stores flat user_id from backend', async () => {
    await useAuthStore.getState().setAuth('tok_12345', {
      id: 'u_1',
      name: 'Raj',
      role: 'driver',
      email: 'driver@avandab.com',
    });
    const s = useAuthStore.getState();
    expect(s.token).toBe('tok_12345');
    expect(s.user?.id).toBe('u_1');
    expect(s.user?.role).toBe('driver');
    expect(s.user?.role).not.toBe('DRIVER'); // not uppercase legacy union
    expect(s.isAuthenticated).toBe(true);
  });

  test('setDriverId updates user profile', async () => {
    await useAuthStore.getState().setAuth('tok_12345', {
      id: 'u_1',
      name: 'Raj',
      role: 'driver',
      email: 'driver@avandab.com',
    });
    useAuthStore.getState().setDriverId('drv_12');
    expect(useAuthStore.getState().user?.driverId).toBe('drv_12');
  });

  test('setDriverId does nothing if user is null', () => {
    useAuthStore.getState().setDriverId('drv_12');
    expect(useAuthStore.getState().user).toBeNull();
  });

  test('logout clears user and token state', async () => {
    await useAuthStore.getState().setAuth('tok_12345', {
      id: 'u_1',
      name: 'Raj',
      role: 'driver',
      email: 'driver@avandab.com',
    });
    await useAuthStore.getState().logout();
    const s = useAuthStore.getState();
    expect(s.token).toBeNull();
    expect(s.user).toBeNull();
    expect(s.isAuthenticated).toBe(false);
  });

  test('loadSession populates state when token and user are in SecureStore', async () => {
    const mockUser = {
      id: 'u_saved',
      name: 'Saved User',
      role: 'driver',
      email: 'saved@avandab.com',
    };
    (SecureStore.getItemAsync as jest.Mock).mockImplementation(async (key: string) => {
      if (key === 'auth_token') return 'saved_token_999';
      if (key === 'auth_user') return JSON.stringify(mockUser);
      return null;
    });

    await useAuthStore.getState().loadSession();
    const s = useAuthStore.getState();
    expect(s.token).toBe('saved_token_999');
    expect(s.user?.id).toBe('u_saved');
    expect(s.isAuthenticated).toBe(true);
    expect(s.isLoading).toBe(false);
  });

  test('loadSession handles missing items or JSON parse errors gracefully', async () => {
    (SecureStore.getItemAsync as jest.Mock).mockImplementation(async () => null);

    await useAuthStore.getState().loadSession();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().isLoading).toBe(false);

    (SecureStore.getItemAsync as jest.Mock).mockRejectedValue(new Error('Storage failure'));
    await useAuthStore.getState().loadSession();
    expect(useAuthStore.getState().isLoading).toBe(false);
  });
});
