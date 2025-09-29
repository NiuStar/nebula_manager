import { reactive } from 'vue';
import { login as apiLogin, logout as apiLogout, getProfile } from '../api';

const state = reactive({
  user: null,
  initialized: false,
  loading: false,
  error: null
});

let sessionPromise = null;

export function useAuth() {
  const ensureSession = async () => {
    if (state.initialized) {
      return;
    }
    if (!sessionPromise) {
      sessionPromise = (async () => {
        try {
          const res = await getProfile();
          state.user = res.data?.data ?? null;
        } catch (err) {
          state.user = null;
        } finally {
          state.initialized = true;
          sessionPromise = null;
        }
      })();
    }
    await sessionPromise;
  };

  const login = async (username, password) => {
    state.loading = true;
    state.error = null;
    try {
      const res = await apiLogin({ username, password });
      state.user = res.data?.data ?? { username };
      state.initialized = true;
      return true;
    } catch (err) {
      state.user = null;
      state.error = err?.response?.data?.error || '登录失败，请检查账户或网络';
      return false;
    } finally {
      state.loading = false;
    }
  };

  const logout = async () => {
    try {
      await apiLogout();
    } catch (error) {
      // ignore network errors during logout
    } finally {
      state.user = null;
      state.initialized = true;
    }
  };

  const clearError = () => {
    state.error = null;
  };

  return {
    state,
    ensureSession,
    login,
    logout,
    clearError
  };
}
