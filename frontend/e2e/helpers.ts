import { Page, expect } from "@playwright/test";

export async function loginAs(page: Page, email: string, password: string) {
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: /enter portal/i }).click();
  await expect(page).not.toHaveURL(/\/login$/);
}

export const ADMIN = { email: "admin@stmaryskabete.ac.ke", password: "Admin@2026" };
export const FINANCE = { email: "finance@stmaryskabete.ac.ke", password: "Finance@2026" };

// The app confirms every mutating action through a custom in-page modal
// (useConfirm/ConfirmDialog), not the native browser confirm(). Several
// confirm buttons share exact text with their trigger button ("Approve",
// "Issue book"), so this scopes to the modal overlay itself rather than
// matching by text+position, which would be a Playwright strict-mode
// violation whenever both are visible at once.
export async function confirmModal(page: Page, label: string) {
  await page.locator("div.fixed.inset-0.z-50").getByRole("button", { name: label, exact: true }).click();
}
