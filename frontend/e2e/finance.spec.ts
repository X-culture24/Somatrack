import { test, expect } from "@playwright/test";
import { loginAs, FINANCE } from "./helpers";

test.describe("Finance", () => {
  test.beforeEach(({ page }) => page.on("dialog", (d) => d.accept()));

  test("send fee reminders logs a result", async ({ page }) => {
    await loginAs(page, FINANCE.email, FINANCE.password);
    await page.goto("/finance");

    // Before this pass FeeReminderLog/NotificationLog existed with nothing
    // ever populating them — send_overdue_reminders() is the automation
    // this button now triggers.
    await page.getByRole("button", { name: "Send fee reminders" }).click();
    await page.getByRole("button", { name: "Send reminders", exact: true }).click();

    await expect(page.getByText(/Reminded \d+ learner/)).toBeVisible({ timeout: 10000 });
  });
});
