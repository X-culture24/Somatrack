import { test, expect } from "@playwright/test";
import { mockApi, signIn, type Mock } from "./mock";

const ASSESSMENT = {
  id: "a-1", name: "CAT 1", assessment_type: "cat1", class_id: "c-west", class_name: "Grade 4 West",
  subject_id: "sub-math", subject_name: "Mathematics", term_id: "t-1", term_name: "Term 1",
  exam_thread_id: null, exam_thread_name: "", total_marks: 30, weight: 100, assessment_date: null, marks_count: 1, is_locked: false,
};
const SHEET = [
  { student_id: "s-brian", full_name: "Brian Kamau", admission_no: "ADM-002", score: 18, percentage: 60, grade_letter: "ME", competency_level: "ME", comments: "" },
  { student_id: "s-zawadi", full_name: "Zawadi Njeri", admission_no: "ADM-003", score: null, percentage: null, grade_letter: "", competency_level: "", comments: "" },
];

async function setup(page: import("@playwright/test").Page, assessment = ASSESSMENT): Promise<Mock> {
  const api = await mockApi(page);
  api.on("GET", "/classes/", { results: [{ id: "c-west", name: "Grade 4 West" }] });
  api.on("GET", "/assessments/", { results: [assessment] });
  api.on("GET", "/assessments/a-1/marksheet/", { assessment, results: SHEET });
  await page.goto("/teacher/marks");
  await page.getByRole("combobox").first().selectOption("c-west");
  await page.getByRole("combobox").nth(1).selectOption("a-1");
  await expect(page.getByText("Brian Kamau")).toBeVisible();
  return api;
}

test.beforeEach(async ({ page }) => { await signIn(page, "teacher"); });

test("shows live percentages and saves only changed rows in one request", async ({ page }) => {
  const api = await setup(page);
  expect(api.calls("GET", "/assessments/")[0].url.searchParams.get("class_id")).toBe("c-west");
  api.on("PUT", "/assessments/a-1/marks/", { saved: 1, cleared: 0 });

  const zawadi = page.locator("tr", { hasText: "Zawadi Njeri" });
  await zawadi.locator("input[type=number]").fill("27");
  await expect(zawadi).toContainText("90.0%");
  await expect(page.getByRole("button", { name: "Save 1 change" })).toBeEnabled();
  await page.getByRole("button", { name: "Save 1 change" }).click();
  await expect(page.getByText("Saved 1 mark.")).toBeVisible();

  const puts = api.calls("PUT", "/assessments/a-1/marks/");
  expect(puts).toHaveLength(1);
  expect(puts[0].body).toEqual({ marks: [{ student_id: "s-zawadi", score: 27, competency_level: "", comments: "" }] });
  expect(api.unhandled).toEqual([]);
});

test("row errors from the server are shown on the right learner", async ({ page }) => {
  const api = await setup(page);
  api.on("PUT", "/assessments/a-1/marks/", () => ({ status: 400, json: { error: "validation", message: "Please correct the highlighted fields.", fields: { "s-brian": "score must be between 0 and 30" } } }));
  await page.locator("tr", { hasText: "Brian Kamau" }).locator("input[type=number]").fill("31");
  await page.getByRole("button", { name: "Save 1 change" }).click();
  await expect(page.locator("tr", { hasText: "Brian Kamau" })).toContainText("score must be between 0 and 30");
});

test("clearing a score sends null", async ({ page }) => {
  const api = await setup(page);
  api.on("PUT", "/assessments/a-1/marks/", { saved: 0, cleared: 1 });
  await page.locator("tr", { hasText: "Brian Kamau" }).locator("input[type=number]").fill("");
  await page.getByRole("button", { name: "Save 1 change" }).click();
  await expect(page.getByText("cleared 1")).toBeVisible();
  expect(api.calls("PUT", "/assessments/a-1/marks/")[0].body.marks[0]).toMatchObject({ student_id: "s-brian", score: null });
});

test("a locked exam thread makes the marksheet read-only", async ({ page }) => {
  await setup(page, { ...ASSESSMENT, is_locked: true, exam_thread_name: "Term 1 2026" });
  await expect(page.getByText("Locked — marks are final")).toBeVisible();
  await expect(page.locator("tbody input[type=number]").first()).toBeDisabled();
  await expect(page.getByRole("button", { name: /Save/ })).toHaveCount(0);
});

test("new assessment form posts the class, subject and 'out of'", async ({ page }) => {
  const api = await mockApi(page);
  api.on("GET", "/classes/", { results: [{ id: "c-west", name: "Grade 4 West" }] });
  api.on("GET", "/assessments/", { results: [] });
  api.on("GET", "/subjects/", { results: [{ id: "sub-math", name: "Mathematics" }] });
  api.on("GET", "/exam-threads/", { results: [{ id: "th-1", name: "Term 1 2026" }], count: 1, page: 1, page_size: 100 });
  api.on("POST", "/assessments/", () => ({ status: 201, json: { ...ASSESSMENT, id: "a-new" } }));
  api.on("GET", "/assessments/a-new/marksheet/", { assessment: { ...ASSESSMENT, id: "a-new" }, results: SHEET });
  await page.goto("/teacher/marks");
  await page.getByRole("combobox").first().selectOption("c-west");
  await page.getByTitle("New assessment").click();
  await page.getByPlaceholder("e.g. Maths CAT 1").fill("Maths CAT 1");
  const form = page.locator("form");
  await form.locator("select").nth(0).selectOption("sub-math");
  await form.locator("select").nth(2).selectOption("th-1");
  await form.locator("input[type=number]").fill("30");
  await form.getByRole("button", { name: "Create assessment" }).click();
  await expect(page.getByText("out of 30")).toBeVisible();
  expect(api.calls("POST", "/assessments/")[0].body).toEqual({
    name: "Maths CAT 1", assessment_type: "cat1", class_id: "c-west", subject_id: "sub-math", total_marks: 30, exam_thread_id: "th-1",
  });
  expect(api.calls("GET", "/exam-threads/")[0].url.searchParams.get("is_locked")).toBe("false");
});
