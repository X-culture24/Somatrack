import { api } from "./client";

export const authApi = {
  login: (email: string, password: string) =>
    api.post("/auth/login/", { email, password }),
  me: () => api.get("/auth/me/"),
  refresh: (refresh: string) =>
    api.post("/auth/refresh/", { refresh }),
};
