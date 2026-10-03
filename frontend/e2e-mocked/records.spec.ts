import { test, expect } from "@playwright/test";
import { mockApi, signIn } from "./mock";

const SCHEMA = {
  name: "discipline-cases", title: "Discipline case", module: "Students", can_write: true,
  fields: [
    { name: "student_id", label: "Student", type: "ref", required: true, ref: { resource: "students" }, filterable: true, in_list: true },
    { name: "incident_date", label: "Incident date", type: "date", required: true, in_list: true },
    { name: "severity", label: "Severity", type: "enum", options: ["minor", "major", "severe"], filterable: true, in_list: true },
    { name: "offence", label: "Offence", type: "text", required: true, searchable: true, in_list: true, max_length: 200 },
    { name: "parent_notified", label: "Parent notified", type: "bool", in_list: true },
    { name: "fine", label: "Fine (KES)", type: "decimal", in_list: true },
  ],
};

const ROW = {
  id: "row-1", created_at: "2026-02-02T08:00:00Z", student_id: "s-brian", student_id_label: "Brian Kamau (ADM-002)",
  incident_date: "2026-02-02", severity: "major", offence: "Missing homework", parent_notified: false, fine: 1500,
};

test.beforeEach(async ({ page }) => { await signIn(page, "admin"); });

test("index groups every register by department", async ({ page }) => {
  const api = await mockApi(page);
  api.on("GET", "/meta/resources/", { results: [
    { name: "visitors", title: "Visitor", module: "Front office", can_write: true },
    { name: "discipline-cases", title: "Discipline case", module: "Students", can_write: true },
    { name: "expenses", title: "Expense", module: "Procurement", can_write: false },
  ] });
  await page.goto("/records");
  await expect(page.getByRole("heading", { name: "Students" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Front office" })).toBeVisible();
  await expect(page.getByRole("link", { name: /Expense.*view only/ })).toBeVisible();
  // Module order follows the school's left rail: Students before Front office.
  const headings = await page.locator("h2").allTextContents();
  expect(headings.indexOf("Students")).toBeLessThan(headings.indexOf("Front office"));
  expect(api.unhandled).toEqual([]);
});

test("table, filters and search are built from the schema", async ({ page }) => {
  const api = await mockApi(page);
  api.on("GET", "/meta/resources/discipline-cases/", SCHEMA);
  api.on("GET", "/meta/resources/discipline-cases/options/student_id/", { results: [{ id: "s-brian", label: "Brian Kamau (ADM-002)" }] });
  api.on("GET", "/discipline-cases/", { results: [ROW], count: 1, page: 1, page_size: 25 });
  await page.goto("/records/discipline-cases");

  const row = page.locator("tbody tr").first();
  await expect(row).toContainText("Brian Kamau (ADM-002)"); // ref shows its label, not the id
  await expect(row).toContainText("Major");                  // enum humanised
  await expect(row).toContainText("KES 1,500");               // money column
  await expect(row).toContainText("No");                      // bool badge

  await page.getByRole("combobox", { name: "Severity" }).selectOption("major");
  await page.getByPlaceholder("Search…").fill("homework");
  await expect.poll(() => api.calls("GET", "/discipline-cases/").at(-1)?.url.search).toContain("severity=major");
  expect(api.calls("GET", "/discipline-cases/").at(-1)?.url.searchParams.get("search")).toBe("homework");
  expect(api.unhandled).toEqual([]);
});

test("add form sends typed values and shows per-field server errors", async ({ page }) => {
  const api = await mockApi(page);
  api.on("GET", "/meta/resources/discipline-cases/", SCHEMA);
  api.on("GET", "/meta/resources/discipline-cases/options/student_id/", { results: [{ id: "s-brian", label: "Brian Kamau (ADM-002)" }] });
  api.on("GET", "/discipline-cases/", { results: [], count: 0, page: 1, page_size: 25 });
  let attempt = 0;
  api.on("POST", "/discipline-cases/", () => (++attempt === 1
    ? { status: 400, json: { error: "validation", message: "Please correct the highlighted fields.", fields: { offence: "must be at most 200 characters" } } }
    : { status: 201, json: ROW }));
  await page.goto("/records/discipline-cases");

  await page.getByRole("button", { name: "Add discipline case" }).click();
  const form = page.locator("form");
  await form.getByRole("combobox", { name: "Student" }).selectOption("s-brian");
  await form.locator("input[type=date]").fill("2026-02-02");
  await form.getByLabel("Offence *").fill("Missing homework");
  await form.getByLabel("Severity").selectOption("major");
  await form.getByLabel("Parent notified").check();
  await form.getByLabel("Fine (KES)").fill("1500");
  await form.getByRole("button", { name: "Save" }).click();

  await expect(form.getByText("must be at most 200 characters")).toBeVisible();
  expect(api.calls("POST", "/discipline-cases/")[0].body).toEqual({
    student_id: "s-brian", incident_date: "2026-02-02", severity: "major", offence: "Missing homework", parent_notified: true, fine: "1500",
  });

  await form.getByRole("button", { name: "Save" }).click();
  await expect(page.locator("form")).toHaveCount(0); // closes on success
  expect(api.unhandled).toEqual([]);
});

test("edit sends only the changed fields", async ({ page }) => {
  const api = await mockApi(page);
  api.on("GET", "/meta/resources/discipline-cases/", SCHEMA);
  api.on("GET", /^\/meta\/resources\/discipline-cases\/options\/student_id\/$/, { results: [{ id: "s-brian", label: "Brian Kamau (ADM-002)" }] });
  api.on("GET", "/discipline-cases/", { results: [ROW], count: 1, page: 1, page_size: 25 });
  api.on("PATCH", "/discipline-cases/row-1/", { ...ROW, severity: "severe" });
  await page.goto("/records/discipline-cases");
  await page.getByTitle("Edit").click();
  await page.locator("form").getByLabel("Severity").selectOption("severe");
  await page.locator("form").getByRole("button", { name: "Save" }).click();
  await expect.poll(() => api.calls("PATCH", "/discipline-cases/row-1/").length).toBe(1);
  expect(api.calls("PATCH", "/discipline-cases/row-1/")[0].body).toEqual({ severity: "severe" });
});

test("view-only users get no add, edit or delete", async ({ page }) => {
  const api = await mockApi(page);
  api.on("GET", "/meta/resources/discipline-cases/", { ...SCHEMA, can_write: false });
  api.on("GET", "/discipline-cases/", { results: [ROW], count: 1, page: 1, page_size: 25 });
  await page.goto("/records/discipline-cases");
  await expect(page.locator("tbody tr")).toContainText("Missing homework");
  await expect(page.getByRole("button", { name: /Add/ })).toHaveCount(0);
  await expect(page.getByTitle("Edit")).toHaveCount(0);
  await expect(page.getByTitle("Delete")).toHaveCount(0);
  expect(api.unhandled).toEqual([]); // and never asks for ref options it may not see
});
