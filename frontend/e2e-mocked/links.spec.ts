import { test, expect } from "@playwright/test";
import { BRIAN, mockApi, signIn } from "./mock";

const TOKEN = "AQEBAAAAACc9VZX0ikZEmaiM2IYhtfYAAAAAAAGGoHjonI_4ocDo7mG4pD";
const CODE = "Q2xhdWRlIHdhcyBoZXJlIQ";

test.describe("QR verification page", () => {
  test("reads the token from the fragment and POSTs it — never in a URL", async ({ page }) => {
    await signIn(page, null); // public page
    const api = await mockApi(page);
    api.on("POST", "/verify/", { valid: true, document: "Payment receipt", school: "ACK St. Mary's School Kabete",
      number: "MPESA-QK1001", amount: 4000, date: "24 Sep 2026", student_initials: "B. K.", status: "genuine" });
    await page.goto("/verify#" + TOKEN);

    await expect(page.getByText("Genuine document")).toBeVisible();
    await expect(page.getByText("MPESA-QK1001")).toBeVisible();
    await expect(page.getByText("KES 4,000")).toBeVisible();
    await expect(page.getByText("B. K.")).toBeVisible();
    const call = api.calls("POST", "/verify/")[0];
    expect(call.body).toEqual({ token: TOKEN });
    expect(call.url.search + call.url.pathname).not.toContain(TOKEN);
  });

  test("shows why a code is not valid", async ({ page }) => {
    await signIn(page, null);
    const api = await mockApi(page);
    api.on("POST", "/verify/", { valid: false, reason: "The amount on record no longer matches this receipt. Contact the school bursar." });
    await page.goto("/verify#" + TOKEN);
    await expect(page.getByText("Not valid")).toBeVisible();
    await expect(page.getByText("no longer matches")).toBeVisible();
  });

  test("tells the user when they're rate limited", async ({ page }) => {
    await signIn(page, null);
    const api = await mockApi(page);
    api.on("POST", "/verify/", () => ({ status: 429, json: { message: "too many requests" } }));
    await page.goto("/verify#" + TOKEN);
    await expect(page.getByText(/Too many checks/)).toBeVisible();
  });
});

test.describe("SMS link page", () => {
  test("signed in: resolves the code and opens the receipt, dropping the code from history", async ({ page }) => {
    await signIn(page, "parent");
    const api = await mockApi(page);
    api.on("POST", "/links/resolve/", { purpose: "receipt", record_id: "p-2", target: "/parent/fees?receipt=p-2" });
    api.on("GET", "/dashboard/", { children: [BRIAN] });
    api.on("GET", "/invoices/", { data: [] });
    api.on("GET", "/payments/", { data: [] });
    await page.goto("/r#" + CODE);
    await expect(page).toHaveURL(/\/parent\/fees\?receipt=p-2$/);
    expect(api.calls("POST", "/links/resolve/")[0].body).toEqual({ code: CODE });
    expect(await page.evaluate(() => sessionStorage.getItem("stmarys-pending-link"))).toBeNull();
  });

  test("signed out: goes to login and keeps the code for afterwards", async ({ page }) => {
    await signIn(page, null);
    const api = await mockApi(page);
    await page.goto("/r#" + CODE);
    await expect(page).toHaveURL(/\/login$/);
    expect(await page.evaluate(() => sessionStorage.getItem("stmarys-pending-link"))).toBe(CODE);
    expect(api.calls("POST", "/links/resolve/")).toHaveLength(0);
  });

  for (const [status, text] of [[403, /belongs to a different parent account/], [404, /expired or is no longer valid/]] as const) {
    test(`a ${status} from the server gives a clear message`, async ({ page }) => {
      await signIn(page, "parent");
      const api = await mockApi(page);
      api.on("POST", "/links/resolve/", () => ({ status, json: { message: "no" } }));
      await page.goto("/r#" + CODE);
      await expect(page.getByText(text)).toBeVisible();
      // A failed link isn't retried on the next sign-in.
      expect(await page.evaluate(() => sessionStorage.getItem("stmarys-pending-link"))).toBeNull();
    });
  }
});
