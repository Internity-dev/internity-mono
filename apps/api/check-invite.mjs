import { chromium } from 'file:///C:/nvm4w/nodejs/node_modules/playwright/index.mjs'

const browser = await chromium.launch()
const page = await browser.newPage({ viewport: { width: 1280, height: 900 } })
await page.goto('http://localhost:5173/login')
await page.fill('#email', 'coordinator@internity.test')
await page.fill('#password', 'password123')
await page.click('button[type="submit"]')
await page.waitForURL('**/dashboard', { timeout: 15000 })
const tourClose = page.locator('.driver-popover-close-btn')
if (await tourClose.isVisible().catch(() => false)) { await tourClose.click(); await page.waitForTimeout(300) }

await page.goto('http://localhost:5173/admin/users')
await page.waitForTimeout(1000)
await page.getByLabel('Department').click()
await page.waitForTimeout(500)
const opt = page.getByRole('option', { name: 'Rekayasa Perangkat Lunak' })
console.log('option count:', await opt.count())
const box = await opt.boundingBox()
console.log('option box:', JSON.stringify(box))
const vp = page.viewportSize()
console.log('viewport:', JSON.stringify(vp))
await page.screenshot({ path: 'invite-dropdown.png', fullPage: false })
await browser.close()
