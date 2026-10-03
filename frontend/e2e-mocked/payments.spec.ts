import { test, expect } from "@playwright/test";
import { mockApi, signIn, type Mock } from "./mock";

async function openForm(page: import("@playwright/test").Page): Promise<Mock> {
  const api = await mockApi(page);
  api.on("GET", "/payments/", { data: [] });
  api.on("GET", "/students/", { results: [{ id: "s-faith", full_name: "Faith Otieno", admission_no: "ADM-001", class_name: "Grade 4 East" }] });
  await page.goto("/finance/payments");
  await page.getByRole("button", { name: "Record a payment" }).click();
  await page.getByPlaceholder("Type a name or admission number…").fill("Fa");
  await page.locator("form select").first().selectOption("s-faith");
  return api;
}

test.beforeEach(async ({ page }) => { await signIn(page, "finance"); });

test("records a cheque with its number and bank, once per form", async ({ page }) => {
  const api = await openForm(page);
  expect(api.calls("GET", "/students/").at(-1)?.url.searchParams.get("search")).toBe("Fa");
  api.on("POST", "/finance/manual-payment/", () => ({ status: 201, json: { payment_id: "p-9", receipt_no: "RCT-20260925-6SHZAZ", duplicate: false } }));

  await page.getByLabel("Amount (KES) *").fill("2500");
  await page.getByLabel("Paid by").selectOption("cheque");
  await page.getByLabel("Cheque no. *").fill("000123");
  await page.getByPlaceholder("e.g. Equity Bank").fill("Equity Bank");
  await page.getByRole("button", { name: /Record payment/ }).click();
  await expect(page.getByText("Recorded RCT-20260925-6SHZAZ. The family has been sent a receipt link.")).toBeVisible();

  const sent = api.calls("POST", "/finance/manual-payment/")[0].body;
  expect(sent).toMatchObject({ student_id: "s-faith", amount: 2500, payment_method: "cheque", cheque_no: "000123", bank_name: "Equity Bank" });
  expect(sent.idempotency_key).toMatch(/^[0-9a-f-]{36}$/);
  expect(api.unhandled).toEqual([]);
});

test("a retry of the same form reuses its idempotency key; a new form gets a new one", async ({ page }) => {
  const api = await openForm(page);
  let n = 0;
  api.on("POST", "/finance/manual-payment/", () => (++n === 1
    ? { status: 502, json: { message: "gateway timeout" } }
    : { status: 201, json: { payment_id: "p-9", receipt_no: "RCT-1", duplicate: false } }));
  await page.getByLabel("Amount (KES) *").fill("1000");
  await page.getByRole("button", { name: /Record payment/ }).click();
  await expect(page.getByText("gateway timeout")).toBeVisible();
  await page.getByRole("button", { name: /Record payment/ }).click();
  await expect(page.getByText(/Recorded RCT-1/)).toBeVisible();
  const [a, b] = api.calls("POST", "/finance/manual-payment/").map((c) => c.body.idempotency_key);
  expect(a).toBe(b); // same form = same payment

  await page.getByPlaceholder("Type a name or admission number…").fill("Fa");
  await page.locator("form select").first().selectOption("s-faith");
  await page.getByLabel("Amount (KES) *").fill("500");
  await page.getByRole("button", { name: /Record payment/ }).click();
  await expect.poll(() => api.calls("POST", "/finance/manual-payment/").length).toBe(3);
  expect(api.calls("POST", "/finance/manual-payment/")[2].body.idempotency_key).not.toBe(a);
});

test("shows field errors from the server", async ({ page }) => {
  const api = await openForm(page);
  api.on("POST", "/finance/manual-payment/", () => ({ status: 400, json: { error: "validation", message: "Please correct the highlighted fields.",
    fields: { reference: "the bank slip or transaction number is required" } } }));
  await page.getByLabel("Amount (KES) *").fill("100");
  await page.getByLabel("Paid by").selectOption("bank_transfer");
  // Browser validation requires the slip number; submit the form directly to reach the server check.
  await page.locator("form").evaluate((f: HTMLFormElement) => f.querySelectorAll("[required]").forEach((el) => el.removeAttribute("required")));
  await page.getByRole("button", { name: /Record payment/ }).click();
  await expect(page.getByText("the bank slip or transaction number is required")).toBeVisible();
});
