import type { Page, Request } from "@playwright/test";

// A tiny in-browser API: tests register handlers by method + path, every
// /api call is answered from them, and every request is recorded so tests
// can assert on exactly what the page sent. Unhandled calls get a 599 and
// are listed in `unhandled`, which each test checks is empty.

export type Handler = (req: Request, body: any, url: URL) => { status?: number; json?: unknown; body?: string; contentType?: string } | undefined;

export type Mock = {
  /** h is a handler, or a plain value sent as a 200 JSON body. For any other status use a handler. */
  on(method: string, path: string | RegExp, h: Handler | unknown): void;
  calls(method: string, path: string | RegExp): { url: URL; body: any }[];
  unhandled: string[];
};

type Role = "admin" | "teacher" | "parent" | "finance";

const USERS: Record<Role, { id: string; email: string; full_name: string; role: string; portal: string }> = {
  admin: { id: "u-admin", email: "admin@it", full_name: "Test Admin", role: "SYSTEM_ADMIN", portal: "admin" },
  teacher: { id: "u-teacher", email: "teacher@it", full_name: "Class Teacher", role: "CLASS_TEACHER", portal: "teacher" },
  parent: { id: "u-parent", email: "parent@it", full_name: "Mary Kamau", role: "PARENT", portal: "parent" },
  finance: { id: "u-finance", email: "bursar@it", full_name: "Bursar", role: "FINANCE_OFFICER", portal: "finance" },
};

/** Signs the browser in as role (or leaves it signed out with null) before the app loads. */
export async function signIn(page: Page, role: Role | null) {
  const state = role ? { user: USERS[role], access: "test-token", refresh: null } : { user: null, access: null, refresh: null };
  await page.addInitScript((s) => localStorage.setItem("stmarys-auth", JSON.stringify({ state: s, version: 0 })), state);
}

export async function mockApi(page: Page): Promise<Mock> {
  const routes: { method: string; path: string | RegExp; h: Handler }[] = [];
  const log: { method: string; url: URL; body: any }[] = [];
  const unhandled: string[] = [];
  const match = (p: string | RegExp, path: string) => (typeof p === "string" ? p === path : p.test(path));

  // Only real API paths — a "**/api/**" glob would also catch the app's own
  // source modules (/src/api/*.ts) and break it.
  await page.route((u) => u.pathname.startsWith("/api/"), async (route) => {
    const req = route.request();
    const url = new URL(req.url());
    const path = url.pathname.replace(/^\/api/, "");
    let body: any = undefined;
    try { body = req.postDataJSON(); } catch { body = req.postData(); }
    log.push({ method: req.method(), url, body });
    // Last registered wins, so a test can override a default.
    const hit = [...routes].reverse().find((r) => r.method === req.method() && match(r.path, path));
    if (!hit) {
      unhandled.push(`${req.method()} ${path}${url.search}`);
      return route.fulfill({ status: 599, json: { message: "unmocked" } });
    }
    const out = hit.h(req, body, url) ?? {};
    if (out.body !== undefined) {
      return route.fulfill({ status: out.status ?? 200, body: out.body, contentType: out.contentType ?? "application/octet-stream" });
    }
    return route.fulfill({ status: out.status ?? 200, json: out.json ?? {} });
  });

  return {
    on(method, path, h) {
      routes.push({ method, path, h: typeof h === "function" ? (h as Handler) : () => ({ json: h }) });
    },
    calls(method, path) {
      return log.filter((c) => c.method === method && match(path, c.url.pathname.replace(/^\/api/, ""))).map(({ url, body }) => ({ url, body }));
    },
    unhandled,
  };
}

export const FAITH = { student_id: "s-faith", full_name: "Faith Otieno", admission_no: "ADM-001", class_name: "Grade 4 East" };
export const BRIAN = { student_id: "s-brian", full_name: "Brian Kamau", admission_no: "ADM-002", class_name: "Grade 4 West" };
