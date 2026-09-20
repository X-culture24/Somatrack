import axios from "axios";
import { useAuth } from "../store/auth";

// NOVA is built and deployed independently.  Its API origin is supplied at
// build time; the Portal keeps the same-origin default for local development.
export const api = axios.create({ baseURL: import.meta.env.VITE_API_BASE_URL || "/api" });

api.interceptors.request.use((config) => {
  const raw = localStorage.getItem("stmarys-auth");
  if (raw) {
    try {
      const { state } = JSON.parse(raw);
      if (state?.access) config.headers.Authorization = `Bearer ${state.access}`;
    } catch { /* ignore */ }
  }
  return config;
});

// Access tokens live for hours, not minutes, but a session left open past
// that point must recover via the refresh token instead of failing forever.
let refreshInFlight: Promise<string> | null = null;

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const { config, response } = error;
    const isAuthEndpoint = config?.url === "/auth/refresh/" || config?.url === "/auth/login/";
    if (response?.status !== 401 || !config || config._retried || isAuthEndpoint) {
      return Promise.reject(error);
    }
    const { refresh } = useAuth.getState();
    if (!refresh) {
      useAuth.getState().logout();
      return Promise.reject(error);
    }
    config._retried = true;
    try {
      if (!refreshInFlight) {
        refreshInFlight = axios
          .post(`${api.defaults.baseURL}/auth/refresh/`, { refresh })
          .then((r) => {
            useAuth.setState({ access: r.data.access });
            return r.data.access as string;
          })
          .finally(() => { refreshInFlight = null; });
      }
      const access = await refreshInFlight;
      config.headers.Authorization = `Bearer ${access}`;
      return api(config);
    } catch (refreshError) {
      useAuth.getState().logout();
      return Promise.reject(refreshError);
    }
  },
);

export function unwrap<T>(data: { results?: T[] } | T[]): T[] {
  return Array.isArray(data) ? data : data.results ?? [];
}
