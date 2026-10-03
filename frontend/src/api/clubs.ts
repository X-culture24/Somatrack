import { api, unwrap } from "./client";

export const clubsApi = {
  list: () => api.get("/clubs/", { params: {} }).then((r) => unwrap(r.data)),
  create: (data: unknown) => api.post("/clubs/", data).then((r) => r.data),
};

export const transportApi = {
  routes: () => api.get("/transport-routes/", { params: {} }).then((r) => unwrap(r.data)),
};
