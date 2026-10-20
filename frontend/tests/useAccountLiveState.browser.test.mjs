import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { build } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { chromium } from 'playwright'

const root = fileURLToPath(new URL('../', import.meta.url))
const frameAt = '2026-10-20T12:00:05Z'
const nextFrameAt = '2026-10-20T12:00:09Z'
const resetAt = '2026-10-20T17:00:00Z'

let browser
let harnessScript
let harnessCSS

before(async () => {
  const result = await build({
    configFile: false,
    root,
    logLevel: 'silent',
    plugins: [
      react(),
      tailwindcss(),
      {
        name: 'account-live-harness',
        enforce: 'pre',
        resolveId(id) {
          if (id === 'virtual:account-live-harness') return '\0account-live-harness'
          if (/^(?:\.\.\/|\.\/)+api$/.test(id)) return '\0account-live-api'
        },
        load(id) {
          if (id === '\0account-live-api') {
            return `export const api = new Proxy({}, { get: (_, name) => (...args) => window.__api(name, args) });
              export const getAdminKey = () => "fixture";
              export const resetAdminAuthState = () => {};
              export class AdminAPIError extends Error {}`
          }
          if (id !== '\0account-live-harness') return
          return `import React from 'react';
            import { createRoot } from 'react-dom/client';
            import { MemoryRouter } from 'react-router-dom';
            import Accounts from ${JSON.stringify(path.join(root, 'src/pages/Accounts.tsx'))};
            import { VisibleChannelsProvider } from ${JSON.stringify(path.join(root, 'src/visibleChannels.tsx'))};
            import { ToastProvider } from ${JSON.stringify(path.join(root, 'src/components/ToastProvider.tsx'))};
            import ${JSON.stringify(path.join(root, 'src/i18n.ts'))};
            import ${JSON.stringify(path.join(root, 'src/index.css'))};
            createRoot(document.getElementById('root')).render(
              React.createElement(MemoryRouter, { initialEntries: ['/'] },
                React.createElement(ToastProvider, null,
                  React.createElement(VisibleChannelsProvider, null,
                    React.createElement('main', { style: { padding: '24px', maxWidth: '1400px', margin: '0 auto' } },
                      React.createElement(Accounts))))));`
        },
      },
    ],
    resolve: { alias: { '@': path.join(root, 'src') } },
    define: { 'process.env.NODE_ENV': '"production"' },
    build: {
      write: false,
      minify: false,
      rollupOptions: { input: 'virtual:account-live-harness' },
      lib: { entry: 'virtual:account-live-harness', formats: ['iife'], name: 'AccountLiveHarness' },
    },
  })
  const bundle = Array.isArray(result) ? result[0] : result
  harnessScript = bundle.output.find((item) => item.type === 'chunk').code
  harnessCSS = bundle.output
    .filter((item) => item.type === 'asset' && item.fileName.endsWith('.css'))
    .map((item) => item.source)
    .join('\n')
  browser = await chromium.launch({ headless: true })
})

after(async () => {
  await browser?.close()
})

function accountRow(usage5h) {
  return {
    id: 1,
    name: '实时额度测试账号',
    email: 'live@example.test',
    account_type: 'oauth',
    status: 'active',
    enabled: true,
    health_tier: 'healthy',
    created_at: '2026-10-01T00:00:00Z',
    group_ids: [],
    models: ['gpt-6-astra'],
    detail_loaded: true,
    plan_type: 'plus',
    usage_percent_5h: usage5h,
    reset_5h_at: resetAt,
    codex_5h_usage_updated_at: frameAt,
    usage_percent_7d: 12,
    reset_7d_at: '2026-10-27T00:00:00Z',
    codex_usage_updated_at: '2026-10-20T11:59:00Z',
    active_requests: 0,
    occupied_requests: 0,
    dynamic_concurrency_limit: 10,
  }
}

function liveState({ usage5h = 93, pending = true, frame = frameAt, status = 'active' } = {}) {
  return {
    accounts: {
      '1': {
        active_requests: 1,
        occupied_requests: 1,
        dynamic_concurrency_limit: 1,
        status,
        using_credits: false,
        plan_type: 'plus',
        usage_percent_5h: usage5h,
        reset_5h_at: resetAt,
        codex_5h_usage_updated_at: frame,
        usage_percent_7d: 12,
        reset_7d_at: '2026-10-27T00:00:00Z',
        codex_usage_updated_at: '2026-10-20T11:59:00Z',
        codex_usage_frame_at: frame,
        codex_usage_frame_kind: 'codex.rate_limits',
        quota_guard_pending: pending,
        cooldown_until: null,
        cooldown_reason: '',
      },
    },
    session_slot_buffer_enabled: true,
  }
}

async function fixture(t) {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } })
  page.setDefaultTimeout(8_000)
  t.after(() => page.close())
  const errors = []
  page.on('pageerror', (error) => errors.push(error.message))
  t.after(() => assert.deepEqual(errors, []))
  await page.route('http://account-live.test/**', (route) => route.fulfill({
    contentType: 'text/html',
    body: '<!doctype html><html><head><meta name="viewport" content="width=device-width, initial-scale=1"></head><body><div id="root"></div></body></html>',
  }))
  await page.goto('http://account-live.test/')
  await page.evaluate(({ initialAccount, initialLive, frameAt, nextFrameAt, resetAt }) => {
    localStorage.setItem('lang', 'zh')
    localStorage.setItem('codex2api:accounts:page-mode', 'pool')
    window.__calls = []
    window.__listUsage = 80
    window.__liveState = initialLive
    window.__initialAccount = initialAccount
    window.__api = async (name, args) => {
      window.__calls.push({ name, args })
      if (name === 'getVisibleChannels') return { channels: ['codex'] }
      if (name === 'getAccountsPage') {
        return {
          accounts: [{ ...window.__initialAccount, usage_percent_5h: window.__listUsage }],
          total: 1,
          page: 1,
          page_size: 20,
          facets: { tags: [], email_domains: [] },
          summary: { total: 1, active: 1, rate_limited: 0, error: 0 },
        }
      }
      if (name === 'getAccountLiveState') return window.__liveState
      if (name === 'getAccountPageStats') return { stats: {} }
      if (name === 'getAccountHealthBars') return { buckets: {} }
      if (name === 'getIntelligenceBars') return { bars: {} }
      if (name === 'getChannelMonitorBillingRates') return { items: [] }
      if (name === 'getAPIKeys') return { keys: [] }
      if (name === 'getSettings') return {}
      if (name === 'getProxies') return { proxies: [] }
      throw new Error(`unexpected API ${name}`)
    }
    window.__liveVariants = {
      initial: initialLive,
      paused: {
        ...initialLive,
        accounts: {
          '1': {
            ...initialLive.accounts['1'],
            usage_percent_5h: 95,
            status: 'quota_paused',
            codex_5h_usage_updated_at: nextFrameAt,
            codex_usage_frame_at: nextFrameAt,
            quota_guard_pending: false,
          },
        },
      },
    }
    window.__frameAt = frameAt
    window.__nextFrameAt = nextFrameAt
    window.__resetAt = resetAt
  }, {
    initialAccount: accountRow(80),
    initialLive: liveState(),
    frameAt,
    nextFrameAt,
    resetAt,
  })
  await page.addStyleTag({ content: harnessCSS })
  await page.addScriptTag({ content: harnessScript })
  return page
}

function card(page) {
  return page.locator('.codex-account-card').first()
}

test('live quota updates the rendered card immediately and an older list response cannot roll it back', async (t) => {
  const page = await fixture(t)
  await page.getByRole('button', { name: '卡片', exact: true }).click()
  const usage = card(page).getByRole('progressbar', { name: '5h' })
  await usage.waitFor()
  await page.waitForFunction(() => document.querySelector('[aria-label="5h"]')?.getAttribute('aria-valuetext') === '93.0%')

  assert.equal(await usage.getAttribute('aria-valuetext'), '93.0%')
  await card(page).getByText('等待额度确认', { exact: true }).waitFor()
  await card(page).locator(`time[datetime="${frameAt}"]`).first().waitFor()
  assert.equal(await card(page).getByText('WS 额度帧', { exact: true }).count(), 1)

  await page.evaluate(() => {
    window.__listUsage = 80
  })
  await page.locator('[data-slot="page-header"]').getByRole('button', { name: '刷新', exact: true }).click()
  await page.waitForFunction(() => window.__calls.filter(({ name }) => name === 'getAccountsPage').length >= 2)
  await page.waitForTimeout(200)
  assert.equal(await usage.getAttribute('aria-valuetext'), '93.0%')
  await card(page).getByText('等待额度确认', { exact: true }).waitFor()

  await page.evaluate(() => {
    window.__liveState = window.__liveVariants.paused
  })
  await page.waitForFunction((nextFrameAt) => {
    const bar = document.querySelector('[aria-label="5h"]')
    return bar?.getAttribute('aria-valuetext') === '95.0%' &&
      Boolean(document.querySelector(`.codex-account-card time[datetime="${nextFrameAt}"]`))
  }, nextFrameAt, { timeout: 3_000 })
  await page.waitForTimeout(200)
  assert.match(await card(page).innerText(), /自动暂停/)
  assert.equal(await card(page).getByText('等待额度确认', { exact: true }).count(), 0)
})
