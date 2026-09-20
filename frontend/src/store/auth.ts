import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api } from "../api/client";

export type User = {
  id: number;
  email: string;
  first_name: string;
  last_name: string;
  full_name: string;
  role: string;
  phone: string;
  portal: "admin" | "finance" | "teacher" | "parent" | "student";
};

type AuthState = {
  user: User | null;
  access: string | null;
  refresh: string | null;
  login: (email: string, password: string) => Promise<User>;
  logout: () => void;
};

export const useAuth = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      access: null,
      refresh: null,
      login: async (email, password) => {
        const { data } = await api.post("/auth/login/", { email, password });
        set({ access: data.access, refresh: data.refresh, user: data.user });
        return data.user as User;
      },
      logout: () => set({ user: null, access: null, refresh: null }),
    }),
    { name: "stmarys-auth" },
  ),
);

export function portalHome(portal: User["portal"]) {
  if (portal === "finance") return "/finance";
  if (portal === "teacher") return "/teacher";
  if (portal === "parent") return "/parent";
  if (portal === "student") return "/student";
  return "/admin";
}
