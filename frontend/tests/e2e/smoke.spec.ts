import { expect, test } from '@playwright/test';

test('renders app shell', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByText('Welcome to Modulr')).toBeVisible();
});
