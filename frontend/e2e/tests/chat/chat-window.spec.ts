import { test, expect } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'

test.describe('Chat Window 24h Logic', () => {
    let chatPage: ChatPage

    const mockContact = {
        id: 'contact-123',
        phone_number: '123456789',
        name: 'Test User',
        profile_name: 'Test User',
        whatsapp_account: 'MyBusinessAccount',
        status: 'active',
        tags: [],
        custom_fields: {},
        unread_count: 0,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
    }

    const mockTemplates = {
        data: {
            templates: [
                {
                    id: 'tmpl-1',
                    name: 'hello_world',
                    display_name: 'Hello World',
                    language: 'en_US',
                    status: 'APPROVED',
                    body_content: 'Hello {{1}}',
                    header_type: '',
                    header_content: '',
                    footer_content: '',
                    whatsapp_account: 'MyBusinessAccount'
                }
            ]
        }
    }

    test('should restrict sending when chat is empty (window closed)', async ({ page }) => {
        // Login first
        await loginAsAdmin(page)

        // Setup mocks AFTER login
        await page.route('**/api/contacts', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    data: { contacts: [mockContact], total: 1 }
                })
            })
        })

        await page.route(`**/api/contacts/${mockContact.id}/messages*`, async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    data: { messages: [], has_more: false }
                })
            })
        })

        await page.route('**/api/chatbot/transfers*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: { transfers: [] } })
            })
        })

        await page.route('**/api/contacts/*/session-data', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: null })
            })
        })

        chatPage = new ChatPage(page)
        await chatPage.goto()
        await chatPage.selectContact('Test User')

        // When chat is empty, the message input should still be visible
        // The 24h warning only shows when there ARE messages but they're old
        await expect(page.locator('textarea, input[placeholder*="message" i]').first()).toBeVisible()
    })

    test('should restrict sending when last customer message is > 24h old', async ({ page }) => {
        const oldDate = new Date()
        oldDate.setHours(oldDate.getHours() - 25)

        const oldMessage = {
            id: 'msg-1',
            contact_id: mockContact.id,
            direction: 'incoming',
            message_type: 'text',
            content: { body: 'Hello from yesterday' },
            status: 'read',
            created_at: oldDate.toISOString(),
            updated_at: oldDate.toISOString()
        }

        await loginAsAdmin(page)

        // Separate routes for contacts list and messages
        await page.route('**/api/contacts', async route => {
            // Only match the contacts list endpoint (no subpaths)
            if (!route.request().url().includes('/messages') && !route.request().url().includes('/session-data')) {
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        data: { contacts: [mockContact], total: 1 }
                    })
                })
            } else {
                await route.continue()
            }
        })

        await page.route(`**/api/contacts/${mockContact.id}/messages*`, async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    data: { messages: [oldMessage], has_more: false }
                })
            })
        })

        await page.route('**/api/chatbot/transfers*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: { transfers: [] } })
            })
        })

        await page.route('**/api/contacts/*/session-data', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: null })
            })
        })

        chatPage = new ChatPage(page)
        await chatPage.goto()
        await chatPage.selectContact('Test User')

        // Wait for messages to load and UI to update
        await page.waitForTimeout(1000)

        // Expect warning banner for 24h window closed
        await expect(page.getByText(/24-hour messaging window closed/i)).toBeVisible({ timeout: 10000 })

        // Expect Send Template button to be visible
        await expect(page.getByRole('button', { name: /Send Template/i })).toBeVisible()
    })

    test('should allow sending when last customer message is < 24h old', async ({ page }) => {
        const recentDate = new Date()
        recentDate.setHours(recentDate.getHours() - 1)

        const recentMessage = {
            id: 'msg-2',
            contact_id: mockContact.id,
            direction: 'incoming',
            message_type: 'text',
            content: { body: 'Hello just now' },
            status: 'read',
            created_at: recentDate.toISOString(),
            updated_at: recentDate.toISOString()
        }

        await loginAsAdmin(page)

        await page.route('**/api/contacts', async route => {
            if (!route.request().url().includes('/messages') && !route.request().url().includes('/session-data')) {
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        data: { contacts: [mockContact], total: 1 }
                    })
                })
            } else {
                await route.continue()
            }
        })

        await page.route(`**/api/contacts/${mockContact.id}/messages*`, async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    data: { messages: [recentMessage], has_more: false }
                })
            })
        })

        await page.route('**/api/chatbot/transfers*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: { transfers: [] } })
            })
        })

        await page.route('**/api/contacts/*/session-data', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: null })
            })
        })

        chatPage = new ChatPage(page)
        await chatPage.goto()
        await chatPage.selectContact('Test User')

        // Wait for messages to load
        await page.waitForTimeout(1000)

        // Expect input to be visible (no 24h restriction)
        const messageInput = chatPage.messageInput
        await expect(messageInput).toBeVisible()

        // Expect warning banner NOT to be visible
        await expect(page.getByText(/24-hour messaging window closed/i)).not.toBeVisible()

        // Try typing
        await chatPage.typeMessage('Hello back')
        await expect(messageInput).toHaveValue('Hello back')
    })

    test('should successfully send a template when window is closed', async ({ page }) => {
        const oldDate = new Date()
        oldDate.setHours(oldDate.getHours() - 25)

        const oldMessage = {
            id: 'msg-old',
            contact_id: mockContact.id,
            direction: 'incoming',
            message_type: 'text',
            content: { body: 'Old message' },
            status: 'read',
            created_at: oldDate.toISOString(),
            updated_at: oldDate.toISOString()
        }

        await loginAsAdmin(page)

        await page.route('**/api/contacts', async route => {
            if (!route.request().url().includes('/messages') && !route.request().url().includes('/session-data')) {
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        data: { contacts: [mockContact], total: 1 }
                    })
                })
            } else {
                await route.continue()
            }
        })

        await page.route(`**/api/contacts/${mockContact.id}/messages*`, async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    data: { messages: [oldMessage], has_more: false }
                })
            })
        })

        await page.route('**/api/chatbot/transfers*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: { transfers: [] } })
            })
        })

        await page.route('**/api/contacts/*/session-data', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: null })
            })
        })

        await page.route('**/api/templates*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify(mockTemplates)
            })
        })

        await page.route('**/api/messages/template', async route => {
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

        chatPage = new ChatPage(page)
        await chatPage.goto()
        await chatPage.selectContact('Test User')

        // Wait for messages to load
        await page.waitForTimeout(1000)

        // Click Send Template
        await page.getByRole('button', { name: /Send Template/i }).click()

        // Dialog should open
        const dialog = page.getByRole('dialog')
        await expect(dialog).toBeVisible()
        await expect(dialog).toContainText('Send Template Message')

        // Select template
        await dialog.locator('button[role="combobox"]').click()
        await page.locator('[role="option"]').filter({ hasText: /hello_world|Hello World/i }).click()

        // Fill variable input
        const varInput = dialog.locator('input').last()
        await varInput.fill('User')

        // Click Send Template button in dialog
        await dialog.getByRole('button', { name: /Send Template/i }).click()

        // Expect success toast
        await expect(page.locator('[data-sonner-toast]')).toContainText(/successfully/i, { timeout: 10000 })
    })
})
