import { test, expect } from "@playwright/test";
import { loginAs, ADMIN, confirmModal } from "./helpers";

test.describe("Payroll", () => {
  test.beforeEach(({ page }) => page.on("dialog", (d) => d.accept()));

  test("create a run, generate payslips from staff profiles, approve, and mark paid", async ({ page }) => {
    await loginAs(page, ADMIN.email, ADMIN.password);
    await page.goto("/admin/payroll");

    // Pick a period unlikely to already exist (PayrollRun is unique on month+year).
    const year = String(2000 + Math.floor(Math.random() * 25));
    await page.getByLabel("Year").fill(year);
    await page.getByRole("button", { name: "Create run" }).click();
    await confirmModal(page, "Create");

    const row = page.locator("tr", { hasText: year }).first();
    await expect(row).toBeVisible();
    await row.click();

    // Generate — previously there was no way to populate PayrollEntry rows
    // at all; PayrollEntry.compute() existed but was never called.
    await page.getByRole("button", { name: "Generate payslips" }).click();
    await confirmModal(page, "Generate");

    // The seeded class teacher (Mary Wambui, basic salary 40,000) should now
    // have a computed payslip with a non-zero gross and net.
    const payslipRow = page.locator("tr", { hasText: "Wambui" });
    await expect(payslipRow).toBeVisible({ timeout: 10000 });
    // headers: Staff no., Name, Basic, Gross, PAYE, NSSF, NHIF, Other deductions, Net, Paid
    await expect(payslipRow.locator("td").nth(2)).toContainText("40,000");
    await expect(payslipRow.locator("td").nth(8)).not.toContainText("KES 0");

    await page.getByRole("button", { name: "Approve payroll" }).click();
    await confirmModal(page, "Approve");
    await expect(row.getByText("approved", { exact: true })).toBeVisible();

    await page.getByRole("button", { name: "Mark as paid" }).click();
    await confirmModal(page, "Mark paid");
    await expect(row.getByText("paid", { exact: true })).toBeVisible();
  });
});
