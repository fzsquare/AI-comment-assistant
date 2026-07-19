import { expect, test } from '@playwright/test'

test('merchant switches one synchronized 7/30-day platform dashboard across desktop and mobile', async ({ page }, testInfo) => {
  const consoleErrors: string[] = []
  page.on('console', (message) => {
    if (message.type() === 'error') consoleErrors.push(message.text())
  })

  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.goto('/merchant/login')
  await page.getByRole('button', { name: '进入商家后台' }).click()
  await expect(page).toHaveURL(/\/merchant\/console$/)
  await expect(page.getByRole('heading', { name: '顾客评价转化' })).toBeVisible()
  await expect(page.locator('[data-funnel-stage]')).toHaveCount(3)
  await expect(page.getByTestId('daily-trend')).toBeVisible()
  await expect(page.locator('.trend-panel table')).toHaveCSS('position', 'absolute')
  const accessibleTableBox = await page.locator('.trend-panel table').boundingBox()
  expect(accessibleTableBox?.width).toBeLessThanOrEqual(1)
  expect(accessibleTableBox?.height).toBeLessThanOrEqual(1)
  await expect(page.locator('[data-metric="review_copy"]')).toHaveCount(0)
  await expect(page.getByTestId('daily-trend')).toHaveAttribute('aria-label', /选择平台/)
  await expect(page.getByTestId('daily-trend')).not.toHaveAttribute('aria-label', /复制评价/)
  await expect(page.getByTestId('review-verification')).toContainText('评论结果验证')
  await expect(page.getByTestId('review-verification')).toContainText('平台数据已核对')
  await expect(page.getByTestId('review-verification')).toContainText('本周引导评论占比')
  await expect(page.getByTestId('review-verification')).toContainText('35.8%')

  const desktopColumns = await page.locator('.effect-grid').evaluate((element) => getComputedStyle(element).gridTemplateColumns)
  const [trendWidth, funnelWidth] = desktopColumns.split(' ').map((value) => Number.parseFloat(value))
  expect(trendWidth).toBeGreaterThan(funnelWidth)
  await page.screenshot({ path: testInfo.outputPath('merchant-effect-desktop.png'), fullPage: true })

  const sevenDayVisits = Number((await page.locator('[data-metric="page_view"] strong').textContent())?.replace(/,/g, ''))
  await page.locator('[data-range="30d"]').click()
  await expect(page.locator('[data-range="30d"]')).toHaveAttribute('aria-pressed', 'true')
  await expect.poll(async () => Number((await page.locator('[data-metric="page_view"] strong').textContent())?.replace(/,/g, ''))).toBeGreaterThan(sevenDayVisits)
  const thirtyDayVisits = Number((await page.locator('[data-metric="page_view"] strong').textContent())?.replace(/,/g, ''))
  const allPlatformSelections = Number((await page.locator('[data-metric="platform_select"] strong').textContent())?.replace(/,/g, ''))
  let selectedPlatformTotal = 0
  for (const platformCode of ['meituan', 'dianping', 'xiaohongshu', 'douyin']) {
    await page.locator('#effect-platform').selectOption(platformCode)
    await expect(page.locator('#effect-platform')).toHaveValue(platformCode)
    selectedPlatformTotal += Number((await page.locator('[data-metric="platform_select"] strong').textContent())?.replace(/,/g, ''))
  }
  expect(selectedPlatformTotal).toBe(allPlatformSelections)

  await page.locator('#effect-platform').selectOption('meituan')
  await expect(page.locator('#effect-platform')).toHaveValue('meituan')
  await expect(page.getByText('访问为全店，其他为美团', { exact: false })).toBeVisible()
  await expect(page.locator('[data-metric="page_view"] strong')).toHaveText(String(thirtyDayVisits).replace(/\B(?=(\d{3})+(?!\d))/g, ','))

  await page.setViewportSize({ width: 375, height: 812 })
  const rangeButtonBox = await page.locator('[data-range="30d"]').boundingBox()
  const platformSelectBox = await page.locator('#effect-platform').boundingBox()
  const funnelBox = await page.locator('.funnel-panel').boundingBox()
  const trendBox = await page.locator('.trend-panel').boundingBox()
  const mobileAxisLabelBox = await page.locator('.daily-chart .axis-label').first().boundingBox()
  const mobileDateLabelBoxes = await page.locator('.daily-chart text[y="242"]').evaluateAll((labels) => labels.map((label) => {
    const box = label.getBoundingClientRect()
    return { x: box.x, width: box.width }
  }))
  expect(funnelBox).not.toBeNull()
  expect(trendBox).not.toBeNull()
  expect(rangeButtonBox?.height).toBeGreaterThanOrEqual(44)
  expect(platformSelectBox?.height).toBeGreaterThanOrEqual(44)
  expect(mobileAxisLabelBox?.height).toBeGreaterThanOrEqual(10)
  for (let index = 1; index < mobileDateLabelBoxes.length; index += 1) {
    expect(mobileDateLabelBoxes[index].x - (mobileDateLabelBoxes[index - 1].x + mobileDateLabelBoxes[index - 1].width)).toBeGreaterThanOrEqual(2)
  }
  expect(funnelBox!.y).toBeLessThan(trendBox!.y)
  await page.screenshot({ path: testInfo.outputPath('merchant-effect-mobile.png'), fullPage: true })

  expect(consoleErrors).toEqual([])
})
