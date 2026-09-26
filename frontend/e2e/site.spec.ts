import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";

const pages = [
  "/",
  "/person/90000001",
  "/person/90000001/wehrpflicht",
  "/methodik",
  "/impressum",
  "/datenschutz",
  "/gibt-es-nicht",
];

async function expectNoViolations(page: Page) {
  const results = await new AxeBuilder({ page })
    .withTags([
      "wcag2a",
      "wcag2aa",
      "wcag21a",
      "wcag21aa",
      "wcag22aa",
      "best-practice",
    ])
    .analyze();
  const summary = results.violations.map(
    (v) => `${v.id}: ${v.nodes.map((n) => n.target.join(" ")).join(", ")}`,
  );
  expect(summary).toEqual([]);
}

for (const scheme of ["light", "dark"] as const) {
  test.describe(`${scheme} mode`, () => {
    test.use({ colorScheme: scheme });
    for (const path of pages) {
      test(`${path} has no accessibility violations`, async ({ page }) => {
        await page.goto(path);
        await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
        await expectNoViolations(page);
      });
    }
  });
}

for (const path of pages) {
  test(`${path} does not scroll sideways`, async ({ page }) => {
    await page.goto(path);
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
    const overflow = await page.evaluate(
      () =>
        document.documentElement.scrollWidth -
        document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
  });
}

test("the skip link is the first stop and jumps to the content", async ({
  page,
  isMobile,
}) => {
  test.skip(isMobile, "keyboard navigation is checked on desktop");
  await page.goto("/");
  await page.keyboard.press("Tab");
  const skip = page.getByRole("link", { name: "Zum Inhalt springen" });
  await expect(skip).toBeFocused();
  await expect(skip).toBeVisible();
  await page.keyboard.press("Enter");
  await expect(page.locator("main")).toBeFocused();
});

test("search leads to a person and their timeline", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("searchbox", { name: "Name" }).fill("muestermann");
  await page.getByRole("link", { name: /Max Müstermann/ }).click();
  await expect(
    page.getByRole("heading", { level: 1, name: "Max Müstermann" }),
  ).toBeVisible();
  await expect(page).toHaveTitle("Max Müstermann – Kurswechsel");
  await page.getByRole("link", { name: /Wehrpflicht/ }).click();
  await expect(page.getByText("2 Aussagen, 1 Positionswechsel")).toBeVisible();
});

test("choosing a topic lists people who spoke about it", async ({ page }) => {
  await page.goto("/");
  await page
    .getByRole("combobox", { name: "Thema" })
    .selectOption("wehrpflicht");
  await expect(page).toHaveURL(/thema=wehrpflicht/);
  await page.getByRole("link", { name: /Erika Musterfrau/ }).click();
  await expect(page).toHaveURL("/person/90000001/wehrpflicht");
  await expect(page.getByRole("heading", { level: 1 })).toContainText(
    "Wehrpflicht",
  );
});

test("the timeline shows statements in order and marks the change", async ({
  page,
}) => {
  await page.goto("/person/90000001/wehrpflicht");
  await expect(page.getByText("5 Aussagen, 1 Positionswechsel")).toBeVisible();
  const dates = await page
    .getByRole("article")
    .getByRole("heading", { level: 3 })
    .allTextContents();
  expect(dates).toEqual([
    "27. September 2018",
    "20. November 2020",
    "2. Juni 2022",
    "16. März 2023",
    "16. März 2023",
  ]);
  const changed = page
    .getByRole("article")
    .filter({ hasText: "Positionswechsel" });
  await expect(changed).toHaveCount(1);
  await expect(changed).toContainText("vorher dafür, jetzt dagegen");
  const pdf = page.getByRole("link", { name: /Protokoll als PDF/ }).first();
  await expect(pdf).toHaveAttribute(
    "href",
    "https://dserver.bundestag.de/btp/19/19051.pdf",
  );
});

test("an error can be reported with the keyboard alone", async ({
  page,
  isMobile,
}) => {
  test.skip(isMobile, "keyboard navigation is checked on desktop");
  await page.goto("/person/90000001/wehrpflicht");
  const button = page.getByRole("button", {
    name: "Fehler melden: 27. September 2018",
  });
  await button.focus();
  await page.keyboard.press("Enter");
  const dialog = page.getByRole("dialog", { name: "Fehler melden" });
  await expect(dialog).toBeVisible();
  await expectNoViolations(page);

  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(button).toBeFocused();

  await page.keyboard.press("Enter");
  await dialog
    .getByLabel("Was ist falsch?")
    .fill("Die Einordnung übersieht den Kontext der Rede.");
  await dialog.getByRole("button", { name: "Meldung senden" }).click();
  await expect(dialog.getByRole("status")).toHaveText(/Vielen Dank/);
});

test("touch targets are at least 44 pixels high", async ({ page }) => {
  await page.goto("/person/90000001/wehrpflicht");
  await expect(page.getByRole("article").first()).toBeVisible();
  const small = await page.evaluate(() =>
    [
      ...document.querySelectorAll<HTMLElement>(
        "a, button, input, select, textarea",
      ),
    ]
      .filter((el) => el.offsetParent !== null && !el.closest(".sr-only"))
      // WCAG 2.5.8 exempts links inside running text.
      .filter((el) => getComputedStyle(el).display !== "inline")
      .filter((el) => el.getBoundingClientRect().height < 44)
      .map((el) => el.outerHTML.slice(0, 80)),
  );
  expect(small).toEqual([]);
});

test("unknown API routes and pages fail gracefully", async ({
  page,
  request,
}) => {
  const res = await request.get("/api/gibt-es-nicht");
  expect(res.status()).toBe(404);
  expect(await res.json()).toEqual({ error: "not_found" });
  await page.goto("/person/gibt-es-nicht");
  await expect(page.getByRole("alert")).toHaveText(
    "Diese Person ist nicht in der Datenbank.",
  );
});
