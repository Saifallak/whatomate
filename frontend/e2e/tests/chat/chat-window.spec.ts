import { test, expect } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'

test.describe('Chat Window 24h Logic', () => {
    let chatPage: ChatPage

    test.beforeEach(async ({ page }) => {
        // Mock SSO providers to ensure Login page loads without backend
        await page.route('**/api/auth/sso/providers', async route => {
            await route.fulfill({ status: 200, body: JSON.stringify({ data: [] }) })
        })

        // Mock Login
        await page.route('**/api/auth/login', async route => {
            await route.fulfill({
                status: 200,
                body: JSON.stringify({
                    data: {
                        access_token: 'fake-token',
                        refresh_token: 'fake-refresh',
                        user: {
                            id: 'admin-id',
                            email: 'admin@test.com',
                            full_name: 'Admin',
                            role: { name: 'admin' },
                            organization_id: 'org-id',
                            is_super_admin: true
                        }
                    }
                })
            })
        })

        // Mock Me/Profile
        await page.route('**/api/me', async route => {
            await route.fulfill({
                status: 200,
                body: JSON.stringify({
                    data: {
                        id: 'admin-id',
                        email: 'admin@test.com',
                        full_name: 'Admin',
                        role: { name: 'admin' },
                        organization_id: 'org-id',
                        is_super_admin: true
                    }
                })
            })
        })

        await loginAsAdmin(page)
        chatPage = new ChatPage(page)
    })

    const mockContact = {
        id: 'contact-123',
        phone_number: '123456789',
        profile_name: 'Test User',
        whatsapp_account: 'MyBusinessAccount',
        created_at: new Date().toISOString()
    }

    const mockTemplates = {
        data: [
            {
                name: 'hello_world',
                language: 'en_US',
                status: 'APPROVED',
                body_content: 'Hello {{1}}',
                components: []
            }
        ]
    }

    test('should restrict sending when chat is empty (window closed)', async ({ page }) => {
        // Mock contacts
        await page.route('**/api/contacts*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: [mockContact], meta: { total: 1 } })
            })
        })

        // Mock empty messages
        await page.route(`**/api/contacts/${mockContact.id}/messages*`, async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: [], meta: { total: 0 } })
            })
        })

        await chatPage.goto()
        await chatPage.selectContact('Test User')

        // Expect input to be disabled
        await expect(chatPage.messageInput).toBeDisabled()

        // Expect warning banner
        await expect(page.getByText(/24-hour.*window.*closed/i)).toBeVisible()

        // Expect send button to be disabled or hidden
        await expect(chatPage.sendButton).toBeDisabled()

        // Expect Template button to be visible
        await expect(page.getByRole('button', { name: /Send Template/i })).toBeVisible()
    })

    test('should restrict sending when last customer message is > 24h old', async ({ page }) => {
        const oldDate = new Date()
        oldDate.setHours(oldDate.getHours() - 25) // 25 hours ago

        const oldMessage = {
            id: 'msg-1',
            direction: 'incoming',
            content: 'Hello from yesterday',
            created_at: oldDate.toISOString()
        }

        // Mock contacts
        await page.route('**/api/contacts*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: [mockContact], meta: { total: 1 } })
            })
        })

        // Mock messages
        await page.route(`**/api/contacts/${mockContact.id}/messages*`, async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: [oldMessage], meta: { total: 1 } })
            })
        })

        await chatPage.goto()
        await chatPage.selectContact('Test User')

        // Expect input to be disabled
        await expect(chatPage.messageInput).toBeDisabled()

        // Expect warning banner
        await expect(page.getByText(/24-hour.*window.*closed/i)).toBeVisible()
    })

    test('should allow sending when last customer message is < 24h old', async ({ page }) => {
        const recentDate = new Date()
        recentDate.setHours(recentDate.getHours() - 1) // 1 hour ago

        const recentMessage = {
            id: 'msg-2',
            direction: 'incoming',
            content: 'Hello just now',
            created_at: recentDate.toISOString()
        }

        // Mock contacts
        await page.route('**/api/contacts*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: [mockContact], meta: { total: 1 } })
            })
        })

        // Mock messages
        await page.route(`**/api/contacts/${mockContact.id}/messages*`, async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: [recentMessage], meta: { total: 1 } })
            })
        })

        await chatPage.goto()
        await chatPage.selectContact('Test User')

        // Expect input to be enabled
        await expect(chatPage.messageInput).toBeEnabled()

        // Expect warning banner to be hidden
        await expect(page.getByText(/24-hour.*window.*closed/i)).not.toBeVisible()

        // Try typing
        await chatPage.typeMessage('Hello back')
        await expect(chatPage.messageInput).toHaveValue('Hello back')
    })

    test('should successfully send a template when window is closed', async ({ page }) => {
        // Mock contacts
        await page.route('**/api/contacts*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: [mockContact], meta: { total: 1 } })
            })
        })

        // Mock empty messages (window closed)
        await page.route(`**/api/contacts/${mockContact.id}/messages*`, async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: [], meta: { total: 0 } })
            })
        })

        // Mock Templates API
        await page.route('**/api/templates*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify(mockTemplates)
            })
        })

        // Mock Send Template API
        await page.route('**/api/messages/template', async route => {
            // Verify payload here if needed
            const postData = route.request().postDataJSON()
            expect(postData.contact_id).toBe(mockContact.id)
            expect(postData.template_name).toBe('hello_world')

            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    data: {
                        id: 'msg-template-1',
                        status: 'sent',
                        template_name: 'hello_world'
                    }
                })
            })
        })

        await chatPage.goto()
        await chatPage.selectContact('Test User')

        // Click Send Template
        await page.getByRole('button', { name: /Send Template/i }).click()

        // Dialog should open
        const dialog = page.getByRole('dialog')
        await expect(dialog).toBeVisible()
        await expect(dialog).toContainText('Send Template Message')

        // Select template (assuming Select component usage, might need specific locator)
        // For Select trigger:
        await dialog.locator('button[role="combobox"]').click()
        await page.locator('[role="option"]').filter({ hasText: 'hello_world' }).click()

        // Assuming body content "Hello {{1}}", we expect an input for variable 1
        const varInput = dialog.locator('input[placeholder="{{1}}"]')
        await expect(varInput).toBeVisible()
        await varInput.fill('User')

        // Click Send
        await dialog.getByRole('button', { name: /Send Message/i }).click()

        // Expect success toast
        await expect(page.locator('[data-sonner-toast]')).toContainText(/successfully/i)
    })
})
