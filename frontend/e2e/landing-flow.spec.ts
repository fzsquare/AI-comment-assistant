import { expect, test } from '@playwright/test'

const landingToken = '11111111-1111-4111-8111-111111111111'

test('customer selects a platform, edits a review, and completes copy-and-open', async ({ page }, testInfo) => {
  const consoleErrors: string[] = []
  page.on('console', (message) => {
    if (message.type() === 'error') consoleErrors.push(message.text())
  })

  await page.route('https://www.xiaohongshu.com/**', (route) => route.abort('aborted'))
  await page.goto(`/landing/${landingToken}`)

  await expect(page.getByRole('heading', { level: 1 })).toContainText('巷子里的椒麻鸡')
  await expect(page.getByRole('heading', { name: '选择评价平台' })).toBeVisible()
  await expect(page.locator('textarea')).toHaveCount(0)
  await page.screenshot({ path: testInfo.outputPath('landing-platform-mobile.png'), fullPage: true })

  await page.locator('[data-platform-code="xiaohongshu"]').click()
  await expect(page).toHaveURL(new RegExp(`/landing/${landingToken}/review/xiaohongshu$`))

  const editor = page.getByRole('textbox', { name: '可编辑的评价内容' })
  const actions = page.getByTestId('review-actions')
  await expect(editor).toBeVisible()
  await expect(page.getByText('是否发布由你决定')).toBeVisible()

  await editor.focus()
  await expect(actions).toHaveCSS('position', 'static')
  await editor.fill('服务很热情，团购核销也很顺，符合我的真实体验。')
  await editor.blur()
  await expect(actions).toHaveCSS('position', 'sticky')
  await expect.poll(async () => {
    const editorBox = await editor.boundingBox()
    const actionsBox = await actions.boundingBox()
    if (!editorBox || !actionsBox) return -1
    return actionsBox.y - (editorBox.y + editorBox.height)
  }).toBeGreaterThanOrEqual(-1)

  await page.getByTestId('primary-platform-action').click()
  await expect(page.getByRole('status')).toContainText('已复制，正在打开小红书')
  await expect(page.getByRole('alert')).toHaveCount(0)
  expect(consoleErrors).toEqual([])

  await page.screenshot({ path: testInfo.outputPath('landing-review-mobile.png'), fullPage: true })
})
