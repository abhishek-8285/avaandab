const { test, expect } = require("@playwright/test"); test("simple", async ({ page }) => { await page.goto("/"); });
