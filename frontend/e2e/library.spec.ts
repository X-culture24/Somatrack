import { test, expect } from "@playwright/test";
import { loginAs, ADMIN, confirmModal } from "./helpers";

test.describe("Library", () => {
  test.beforeEach(({ page }) => page.on("dialog", (d) => d.accept()));

  test("issue, renew, and return a loan", async ({ page }) => {
    await loginAs(page, ADMIN.email, ADMIN.password);
    await page.goto("/admin/library");

    await page.getByLabel("Book").selectOption({ index: 1 });
    await page.getByLabel("Borrower").selectOption({ index: 1 });
    await page.getByLabel("Due date").fill("2026-12-01");
    await page.getByRole("button", { name: "Issue book" }).click();
    await confirmModal(page, "Issue book");

    // Checkout must decrement available copies (headers: Title, Author,
    // Level, Copies, Available, Status).
    const bookRow = page.locator("tr", { hasText: "CBC Mathematics Grade 4" });
    await expect(bookRow.locator("td").nth(4)).toHaveText("1", { timeout: 10000 });

    await page.getByRole("button", { name: "loans", exact: true }).click();
    const loanRow = page.locator("tr", { hasText: "CBC Mathematics Grade 4" }).first();
    await expect(loanRow).toBeVisible();

    // Renew — previously wired to nothing in the API; also previously could
    // shorten the due date instead of extending it when called well before
    // the due date (fixed in Loan.renew()).
    await loanRow.getByRole("button", { name: "Renew" }).click();
    await confirmModal(page, "Renew");

    // Return — previously never flipped loan.status to "returned".
    await loanRow.getByRole("button", { name: "Return" }).click();
    await confirmModal(page, "Confirm return");
    await expect(loanRow).toHaveCount(0);
  });
});
