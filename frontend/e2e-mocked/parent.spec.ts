import { test, expect } from "@playwright/test";
import { BRIAN, FAITH, mockApi, signIn } from "./mock";

const HISTORY = {
  student_id: BRIAN.student_id, full_name: BRIAN.full_name, admission_no: BRIAN.admission_no,
  terms: [
    { term_id: "t1", term_name: "Term 1", academic_year: "2026", start_date: "2026-01-05", class_id: "c-west", class_name: "Grade 4 West",
      mean: 59.5, mean_grade: "ME", subjects: [
        { subject_id: "m", subject_name: "Mathematics", percentage: 74, grade: "ME", remarks: "", assessments: 2 },
        { subject_id: "e", subject_name: "English", percentage: 45, grade: "AE", remarks: "", assessments: 1 }] },
    { term_id: "t2", term_name: "Term 2", academic_year: "2026", start_date: "2026-05-04", class_id: "c-east", class_name: "Grade 4 East",
      mean: 91, mean_grade: "EE", subjects: [{ subject_id: "m", subject_name: "Mathematics", percentage: 91, grade: "EE", remarks: "", assessments: 1 }] },
  ],
};

test.use({ viewport: { width: 390, height: 844 } }); // parents are on phones
test.beforeEach(async ({ page }) => { await signIn(page, "parent"); });

test("results trail: newest term first, each with the class sat in, trend and transcript", async ({ page }) => {
  const api = await mockApi(page);
  api.on("GET", "/dashboard/", { children: [BRIAN] });
  api.on("GET", `/results/students/${BRIAN.student_id}/history/`, HISTORY);
  api.on("GET", `/results/students/${BRIAN.student_id}/transcript/`, () => ({ body: "%PDF-1.4 test", contentType: "application/pdf" }));
  await page.goto("/parent/progress");

  const cards = page.locator("ol > li");
  await expect(cards).toHaveCount(2);
  await expect(cards.nth(0)).toContainText("Term 2 2026");
  await expect(cards.nth(0)).toContainText("Grade 4 East");
  await expect(cards.nth(0)).toContainText("▲ 31.5"); // 91 vs 59.5
  await expect(cards.nth(1)).toContainText("Grade 4 West");
  await expect(cards.nth(1)).toContainText("English");

  const [dl] = await Promise.all([page.waitForEvent("download"), page.getByRole("button", { name: /Download academic record/ }).click()]);
  expect(dl.suggestedFilename()).toBe("academic-record-ADM-002.pdf");
  expect(api.unhandled).toEqual([]);
});

test("a parent with two children can switch between them", async ({ page }) => {
  const api = await mockApi(page);
  api.on("GET", "/dashboard/", { children: [FAITH, BRIAN] });
  api.on("GET", `/results/students/${FAITH.student_id}/history/`, { ...HISTORY, student_id: FAITH.student_id, admission_no: FAITH.admission_no, terms: [] });
  api.on("GET", `/results/students/${BRIAN.student_id}/history/`, HISTORY);
  await page.goto("/parent/progress");
  await expect(page.getByText("No results recorded yet.")).toBeVisible();
  await page.getByRole("button", { name: "Brian" }).click();
  await expect(page.locator("ol > li")).toHaveCount(2);
});

test("fees: receipts per child, the one from the SMS highlighted and downloadable", async ({ page }) => {
  const api = await mockApi(page);
  api.on("GET", "/dashboard/", { children: [BRIAN] });
  api.on("GET", "/invoices/", { data: [] });
  api.on("GET", "/payments/", { data: [
    { id: "p-1", receipt_no: "RCT-20260925-AAAAAA", amount: 1500, payment_method: "cash", payment_date: "2026-09-25", status: "confirmed" },
    { id: "p-2", receipt_no: "MPESA-QK1001", amount: 4000, payment_method: "mpesa", payment_date: "2026-09-24", status: "confirmed" },
    { id: "p-3", receipt_no: "MPESA-QK0999", amount: 900, payment_method: "mpesa", payment_date: "2026-09-20", status: "reversed" },
  ] });
  api.on("GET", "/payments/p-2/receipt/", () => ({ body: "%PDF-1.4 test", contentType: "application/pdf" }));
  await page.goto("/parent/fees?receipt=p-2");

  await expect(page.locator("li", { hasText: "KES 4,000" })).toHaveClass(/ring-2/);
  await expect(page.locator("li", { hasText: "KES 1,500" })).not.toHaveClass(/ring-2/);
  await expect(page.getByText("MPESA-QK0999")).toHaveCount(0); // reversed payments aren't receipts
  expect(api.calls("GET", "/payments/")[0].url.searchParams.get("student_id")).toBe(BRIAN.student_id);

  const [dl] = await Promise.all([page.waitForEvent("download"), page.locator("li.ring-2").getByRole("button").click()]);
  expect(dl.suggestedFilename()).toBe("receipt-MPESA-QK1001.pdf");
});
