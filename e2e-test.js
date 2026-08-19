const { test, expect } = require('@playwright/test');

test('e2e test', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveTitle(/Avandab/);
});
