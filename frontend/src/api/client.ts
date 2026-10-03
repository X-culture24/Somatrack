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

// The direct-to-DB resources (students, guardians, academics, clubs...) wrap
// list responses as { results: [...] }; the finance/payroll endpoints
// (backed by finance-svc via RPC) use { data: [...] } instead. Same shape,
// different key — this helper accepts either so callers don't need to know
// which family a given endpoint belongs to.
export function unwrap<T>(data: { results?: T[]; data?: T[] } | T[]): T[] {
  return Array.isArray(data) ? data : data.results ?? data.data ?? [];
}

// Fetches a binary file (PDF) through the authenticated axios instance —
// a plain <a href> can't carry the Bearer token — then saves it via a
// throwaway object URL, the standard way to trigger a browser download
// from JS-fetched bytes.
export async function downloadFile(url: string, filename: string) {
  const res = await api.get(url, { responseType: "blob" });
  const blobUrl = window.URL.createObjectURL(res.data);
  const a = document.createElement("a");
  a.href = blobUrl;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  window.URL.revokeObjectURL(blobUrl);
}
