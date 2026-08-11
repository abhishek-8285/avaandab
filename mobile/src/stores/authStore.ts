import { create } from 'zustand';
import * as SecureStore from 'expo-secure-store';

interface UserSession {
  id: string;
  name: string;
  role: 'DRIVER' | 'DISPATCHER' | 'ADMIN';
  email: string;
}

interface AuthState {
  token: string | null;
  user: UserSession | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  setAuth: (token: string, user: UserSession) => Promise<void>;
  logout: () => Promise<void>;
  loadSession: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  user: null,
  isAuthenticated: false,
  isLoading: true,

  setAuth: async (token, user) => {
    await SecureStore.setItemAsync('auth_token', token);
    await SecureStore.setItemAsync('user_session', JSON.stringify(user));
    set({ token, user, isAuthenticated: true, isLoading: false });
  },

  logout: async () => {
    await SecureStore.deleteItemAsync('auth_token');
    await SecureStore.deleteItemAsync('user_session');
    set({ token: null, user: null, isAuthenticated: false, isLoading: false });
  },

  loadSession: async () => {
    try {
      const token = await SecureStore.getItemAsync('auth_token');
      const userJson = await SecureStore.getItemAsync('user_session');
      if (token && userJson) {
        set({ token, user: JSON.parse(userJson), isAuthenticated: true, isLoading: false });
      } else {
        set({ isLoading: false });
      }
    } catch {
      set({ isLoading: false });
    }
  },
}));
