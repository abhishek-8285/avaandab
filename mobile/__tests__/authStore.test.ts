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
});
