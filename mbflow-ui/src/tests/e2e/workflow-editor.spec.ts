import { test, expect } from '@playwright/test'

test.describe('Workflow Creation and Management', () => {
    test.beforeEach(async ({ page }) => {
        await page.goto('/workflows/new')
        await page.waitForLoadState('networkidle')
    })

    test('should load workflow editor', async ({ page }) => {
        await expect(page).toHaveTitle(/Workflow Editor/)
        await expect(page.locator('.node-palette')).toBeVisible()
        await expect(page.locator('.workflow-canvas')).toBeVisible()
    })

    test('should display node palette categories', async ({ page }) => {
        // Check for category headers
        await expect(page.getByText('Control Flow')).toBeVisible()
        await expect(page.getByText('Transform')).toBeVisible()
        await expect(page.getByText('Integration')).toBeVisible()
        await expect(page.getByText('AI & ML')).toBeVisible()
        await expect(page.getByText('Data')).toBeVisible()
    })

    test('should expand and show nodes in categories', async ({ page }) => {
        // Expand Control Flow category
        await page.getByText('Control Flow').click()
        await expect(page.getByText('Conditional Router')).toBeVisible()
        await expect(page.getByText('Parallel')).toBeVisible()

        // Start and End should NOT be visible
        await expect(page.getByText('Start', { exact: true })).not.toBeVisible()
        await expect(page.getByText('End', { exact: true })).not.toBeVisible()
    })

    test('should add nodes to canvas via drag and drop', async ({ page }) => {
        // Expand Transform category
        await page.getByText('Transform').click()

        // Get the Transform node element
        const transformNode = page.locator('.node-item').filter({ hasText: 'Transform' }).first()

        // Get canvas bounds
        const canvas = page.locator('.workflow-canvas')
        const canvasBounds = await canvas.boundingBox()

        if (!canvasBounds) throw new Error('Canvas not found')

        // Drag Transform node to canvas
        await transformNode.dragTo(canvas, {
            targetPosition: { x: 300, y: 300 }
        })

        // Wait for node to appear
        await page.waitForTimeout(500)

        // Verify node was added (check for vue-flow node)
        const addedNodes = page.locator('.vue-flow__node')
        await expect(addedNodes).toHaveCount(1)
    })

    test('should generate unique names for duplicate node types', async ({ page }) => {
        // Expand Transform category
        await page.getByText('Transform').click()

        const transformNode = page.locator('.node-item').filter({ hasText: 'Transform' }).first()
        const canvas = page.locator('.workflow-canvas')

        // Add first Transform node
        await transformNode.dragTo(canvas, {
            targetPosition: { x: 300, y: 300 }
        })
        await page.waitForTimeout(500)

        // Add second Transform node
        await transformNode.dragTo(canvas, {
            targetPosition: { x: 500, y: 300 }
        })
        await page.waitForTimeout(500)

        // Should have 2 nodes
        const nodes = page.locator('.vue-flow__node')
        await expect(nodes).toHaveCount(2)

        // Click on second node to check its name
        await nodes.nth(1).click()
        await page.waitForTimeout(300)

        // Properties panel should show name with random suffix
        const nameInput = page.locator('input[label="Name"]').or(page.locator('.v-field__input input')).first()
        const nameValue = await nameInput.inputValue()

        // Second node should have format: transform_adjective_noun
        expect(nameValue).toMatch(/^transform_[a-z]+_[a-z]+$/)
    })

    test('should connect nodes with edges', async ({ page }) => {
        // Add two Transform nodes
        await page.getByText('Transform').click()
        const transformNode = page.locator('.node-item').filter({ hasText: 'Transform' }).first()
        const canvas = page.locator('.workflow-canvas')

        await transformNode.dragTo(canvas, { targetPosition: { x: 300, y: 300 } })
        await page.waitForTimeout(500)

        await transformNode.dragTo(canvas, { targetPosition: { x: 600, y: 300 } })
        await page.waitForTimeout(500)

        // Find handles
        const sourceHandle = page.locator('.vue-flow__node').first().locator('.vue-flow__handle-right')
        const targetHandle = page.locator('.vue-flow__node').nth(1).locator('.vue-flow__handle-left')

        // Connect nodes
        await sourceHandle.dragTo(targetHandle)
        await page.waitForTimeout(500)

        // Verify edge was created
        const edges = page.locator('.vue-flow__edge')
        await expect(edges).toHaveCount(1)
    })

    test('should update node name with snake_case formatting', async ({ page }) => {
        // Add a Transform node
        await page.getByText('Transform').click()
        const transformNode = page.locator('.node-item').filter({ hasText: 'Transform' }).first()
        const canvas = page.locator('.workflow-canvas')

        await transformNode.dragTo(canvas, { targetPosition: { x: 300, y: 300 } })
        await page.waitForTimeout(500)

        // Click on node to select it
        await page.locator('.vue-flow__node').first().click()
        await page.waitForTimeout(300)

        // Find name input in properties panel
        const nameInput = page.locator('.v-field__input input').first()

        // Clear and type new name
        await nameInput.clear()
        await nameInput.fill('My Custom Node Name')
        await nameInput.blur()
        await page.waitForTimeout(300)

        // Verify it was converted to snake_case
        const finalValue = await nameInput.inputValue()
        expect(finalValue).toBe('my_custom_node_name')
    })

    test('should save workflow successfully', async ({ page }) => {
        // Add a simple workflow
        await page.getByText('Transform').click()
        const transformNode = page.locator('.node-item').filter({ hasText: 'Transform' }).first()
        const canvas = page.locator('.workflow-canvas')

        await transformNode.dragTo(canvas, { targetPosition: { x: 300, y: 300 } })
        await page.waitForTimeout(500)

        // Click Save button
        const saveButton = page.getByRole('button', { name: /save/i })
        await saveButton.click()

        // Wait for navigation or success indicator
        await page.waitForTimeout(2000)

        // URL should change from /new to a UUID
        const url = page.url()
        expect(url).not.toContain('/new')
        expect(url).toMatch(/\/workflows\/[a-f0-9-]+/)
    })

    test('should delete node from canvas', async ({ page }) => {
        // Add a Transform node
        await page.getByText('Transform').click()
        const transformNode = page.locator('.node-item').filter({ hasText: 'Transform' }).first()
        const canvas = page.locator('.workflow-canvas')

        await transformNode.dragTo(canvas, { targetPosition: { x: 300, y: 300 } })
        await page.waitForTimeout(500)

        // Select node
        await page.locator('.vue-flow__node').first().click()
        await page.waitForTimeout(300)

        // Click delete button in properties panel
        const deleteButton = page.getByRole('button', { name: /delete node/i })
        await deleteButton.click()
        await page.waitForTimeout(500)

        // Verify node was removed
        const nodes = page.locator('.vue-flow__node')
        await expect(nodes).toHaveCount(0)
    })
})

test.describe('Workflow Editor UI', () => {
    test('should have all toolbar buttons', async ({ page }) => {
        await page.goto('/workflows/new')
        await page.waitForLoadState('networkidle')

        await expect(page.getByRole('button', { name: /undo/i })).toBeVisible()
        await expect(page.getByRole('button', { name: /redo/i })).toBeVisible()
        await expect(page.getByRole('button', { name: /fit view/i })).toBeVisible()
        await expect(page.getByRole('button', { name: /auto layout/i })).toBeVisible()
        await expect(page.getByRole('button', { name: /save/i })).toBeVisible()
        await expect(page.getByRole('button', { name: /run/i })).toBeVisible()
    })

    test('should switch between palette and variables tabs', async ({ page }) => {
        await page.goto('/workflows/new')
        await page.waitForLoadState('networkidle')

        // Should start on Palette tab
        await expect(page.getByText('Node Palette')).toBeVisible()

        // Switch to Variables tab
        await page.getByRole('tab', { name: /variables/i }).click()
        await page.waitForTimeout(300)

        // Switch back to Palette
        await page.getByRole('tab', { name: /palette/i }).click()
        await page.waitForTimeout(300)

        await expect(page.getByText('Node Palette')).toBeVisible()
    })
})
