import { test, expect } from "@playwright/test";
import { loginAs, ADMIN, confirmModal } from "./helpers";

test.describe("Admissions", () => {
  // Guards against a stray window.alert() (used by the onError handlers)
  // blocking the page if something unexpectedly fails.
  test.beforeEach(({ page }) => page.on("dialog", (d) => d.accept()));

  test("submit, approve, and enroll an application end-to-end", async ({ page }) => {
    await loginAs(page, ADMIN.email, ADMIN.password);
    await page.goto("/admin/admissions");

    const applicantName = `Test Learner ${Date.now()}`;
    await page.getByLabel("Learner name").fill(applicantName);
    await page.getByLabel("Guardian name").fill("Test Guardian");
    await page.getByLabel("Guardian phone").fill("0700000000");
    await page.getByRole("button", { name: "Submit application" }).click();
    await confirmModal(page, "Submit");

    const row = page.locator("tr", { hasText: applicantName });
    await expect(row).toBeVisible();
    await expect(row.getByText("submitted", { exact: true })).toBeVisible();

    // Approve — a plain status PATCH, unaffected by the enroll() bug below.
    await row.getByRole("button", { name: "Approve" }).click();
    await confirmModal(page, "Approve");
    await expect(row.getByText("approved", { exact: true })).toBeVisible();

    // Enroll — this is the action that previously 500'd: it referenced
    // application.applicant_name / guardian_phone / guardian_name, none of
    // which exist on the model (the real fields are pupil_full_name and
    // parent_guardian_*). There was also no Enroll control in the UI at all
    // before this pass — admins had no way to reach this action.
    await row.getByRole("button", { name: "Enroll" }).click();
    await row.locator("select").selectOption({ index: 1 });
    const admissionNo = `E2E-${Date.now()}`;
    await row.getByPlaceholder("Admission no.").fill(admissionNo);
    await row.getByRole("button", { name: "Enroll" }).click();
    await confirmModal(page, "Enroll");

    await expect(row.getByText("enrolled", { exact: true })).toBeVisible({ timeout: 10000 });

    // The new student should now exist with an auto-generated invoice.
    await page.goto("/admin/students");
    await page.getByPlaceholder(/search/i).first().fill(admissionNo);
    await expect(page.locator("tr", { hasText: admissionNo })).toBeVisible();
  });
});
