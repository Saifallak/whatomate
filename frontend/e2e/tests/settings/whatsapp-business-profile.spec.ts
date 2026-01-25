import { test, expect } from '@playwright/test';

test.describe('WhatsApp Business Profile', () => {
    test.beforeEach(async ({ page }) => {
        // Login as admin
        await page.goto('/login');
        await page.fill('input[type="email"]', 'admin@example.com');
        await page.fill('input[type="password"]', 'password');
        await page.click('button[type="submit"]');
        await expect(page).toHaveURL('/dashboard');
    });

    test('should view business profile', async ({ page }) => {
        await page.goto('/settings/accounts');
        
        // Wait for accounts to load
        await expect(page.locator('h1')).toContainText('WhatsApp Accounts');
        
        // Find the "Store" button (Business Profile) and click it
        // Assuming at least one account exists
        const profileButton = page.locator('button:has(svg.text-emerald-500)').first();
        await expect(profileButton).toBeVisible();
        await profileButton.click();
        
        // Check if dialog opens
        await expect(page.locator('div[role="dialog"]')).toBeVisible();
        await expect(page.locator('h2')).toContainText('Business Profile');
        
        // Check if fields are present
        await expect(page.locator('input#about')).toBeVisible();
        await expect(page.locator('textarea#description')).toBeVisible();
        await expect(page.locator('input#email')).toBeVisible();
        await expect(page.locator('input#address')).toBeVisible();
    });
});
