/**
 * 前端 Mock 层：自定义 axios adapter，VITE_USE_MOCK=true 时启用。
 * 让前端脱离 Go 后端 / MySQL / Python 服务也能独立启动调试（尤其落地页）。
 *
 * 启用方式：`npm run dev:mock`（见 package.json）。
 * 数据在内存里可变，create/delete 会反映到后续 GET，方便交互调试。
 */
import type { AxiosAdapter, AxiosResponse } from 'axios'

const envelope = (data: unknown) => ({ code: 0, message: 'ok', data })
const delay = (ms: number) => new Promise((r) => setTimeout(r, ms))

// 自包含 SVG 占位图（无需联网也能显示）
const ph = (text: string, bg: string) =>
  'data:image/svg+xml;utf8,' +
  encodeURIComponent(
    `<svg xmlns='http://www.w3.org/2000/svg' width='300' height='220'>` +
      `<rect width='100%' height='100%' fill='${bg}'/>` +
      `<text x='50%' y='50%' fill='#fff' font-size='20' font-family='sans-serif' ` +
      `text-anchor='middle' dominant-baseline='middle'>${text}</text></svg>`
  )

let seq = 1000
const nextId = () => ++seq

// ---------------- 可变 mock 状态 ----------------
const store = {
  id: 1,
  uuid: '11111111-1111-4111-8111-111111111111',
  typeId: 1,
  storeName: '巷子里的椒麻鸡（Mock 演示）',
  industryType: '川菜/餐饮',
  storeIntro: '一家适合朋友聚会的本地川菜小馆',
  address: '成都市武侯区科华北路 18 号',
  primaryPlatformStyle: 'xiaohongshu',
  brandTone: '轻松自然',
  status: 1
}

let keywords = [
  { id: 1, keyword: '招牌椒麻鸡', sortNo: 1 },
  { id: 2, keyword: '酸菜鱼', sortNo: 2 },
  { id: 3, keyword: '干锅虾', sortNo: 3 },
  { id: 4, keyword: '环境舒服', sortNo: 4 },
  { id: 5, keyword: '适合聚餐', sortNo: 5 }
]

let images = [
  { id: 1, imageUrl: ph('菜品图 1', '#f59e0b'), thumbnailUrl: ph('菜品图 1', '#f59e0b'), status: 1, sortNo: 1 },
  { id: 2, imageUrl: ph('环境图', '#10b981'), thumbnailUrl: ph('环境图', '#10b981'), status: 1, sortNo: 2 },
  { id: 3, imageUrl: ph('招牌菜', '#6366f1'), thumbnailUrl: ph('招牌菜', '#6366f1'), status: 1, sortNo: 3 }
]

let platformLinks = [
  { id: 1, platformCode: 'xiaohongshu', platformName: '小红书', buttonText: '去小红书发布', targetUrl: 'https://www.xiaohongshu.com', backupUrl: '', sortNo: 1, status: 1 },
  { id: 2, platformCode: 'dianping', platformName: '大众点评', buttonText: '去点评发布', targetUrl: 'https://www.dianping.com', backupUrl: '', sortNo: 2, status: 1 },
  { id: 3, platformCode: 'douyin', platformName: '抖音', buttonText: '去抖音发布', targetUrl: 'https://www.douyin.com', backupUrl: '', sortNo: 3, status: 1 }
]

let merchantReviews = [
  { id: 901, platformStyle: 'xiaohongshu', content: '周五和朋友来的，椒麻鸡不错，麻香不冲。', tags: '招牌椒麻鸡', sourceType: 'ai', status: 'available' }
]

let tasks = [
  { id: 1, storeId: 1, platformStyle: 'xiaohongshu', triggerType: 'manual', targetCount: 10, successCount: 8, failedCount: 2, status: 'success' }
]

let nfcTags = [
  { id: 1, tagCode: 'TAG-DEMO-001', storeId: 1, landingToken: 'mock-demo-001', status: 'bound', remark: '演示标签' }
]

// 多商家：演示「每个商家有自己独立的数据」（管理员能看到全部，商家只看到自己的）
const merchants = [
  { id: 1, account: 'merchant', merchantName: '巷子里的椒麻鸡', contactName: '张三', status: 1 },
  { id: 2, account: 'merchant2', merchantName: '舒缘足道', contactName: '李四', status: 1 }
]
const stores = [
  { ...store, id: 1, merchantUserId: 1 },
  { id: 2, merchantUserId: 2, storeName: '舒缘足道', industryType: '足疗按摩', storeIntro: '', address: '', primaryPlatformStyle: 'dianping', brandTone: '轻松自然', status: 1 }
]

// ---------------- 落地页评价池（已 humanize，{{tag}} 会被替换成顾客选的菜）----------------
const reviewPool: Record<string, string[]> = {
  xiaohongshu: [
    '人均80挖到的小馆，朋友局没踩雷\n\n周五下班拉着同事来的，就想找个能坐下来好好聊的地方。点了{{tag}}，麻味是慢慢上来的那种，不冲，鸡肉也嫩。我们仨边吃边聊坐了快两小时也没人催。就是饭点去得等一小会儿。\n\n#探店 #川菜 #朋友聚餐',
    '本来没抱期待，结果有点惊喜\n\n被同事拉来的，点的{{tag}}分量给得实在，两个人三个菜没吃完。老板还问我们能不能吃辣，挺细心。位置藏在巷子里不太好找，但味道对得起这个价。\n\n#宝藏小店 #本地美食'
  ],
  dianping: [
    '上周三中午和客户来谈事，订了靠窗的位置。点了{{tag}}，上菜挺快，味道稳，服务员还主动帮我们分餐。人均120左右，唯一就是饭点有点吵，整体满意，下次还会再来。',
    '周末带爸妈随便吃的，{{tag}}做得不错，我妈一直夸。环境收拾得干净，座位间距也够。人均不算低但能接受，就是停车稍微远一点。'
  ],
  douyin: [
    '上周带同事去吃过，{{tag}}是真不错，上菜也快！就是饭点人有点多等了会儿。人均80左右，想吃的可以先团个套餐～',
    '视频里没拍到，他家{{tag}}也挺顶的。服务员小姐姐很热情，唯一就是店有点小。住附近的可以去试试。'
  ]
}

// 行业 → 推荐标签（与后端 keyword_suggestions.go 对齐的精简版）
function suggestTagsByIndustry(industryType: string): string[] {
  const t = (industryType || '').toLowerCase()
  const table: Array<[string[], string[]]> = [
    [['足疗', '足浴', '按摩', '推拿', '采耳', '养生'], ['手法专业', '力度合适', '技师专业', '服务热情', '环境安静', '干净卫生']],
    [['理发', '美发', '发型', '剪发', '烫染'], ['发型师专业', '听需求', '剪得满意', '不推办卡', '洗头舒服', '环境好']],
    [['美甲', '美睫', '光疗'], ['款式好看', '手法细致', '卸甲不伤', '持久度好', '环境干净', '服务热情']],
    [['宠物', '狗', '猫'], ['师傅专业', '温柔耐心', '剪得好看', '干净无异味', '宠物不抗拒']],
    [['美容', '护肤', '皮肤'], ['手法专业', '项目效果', '不硬推卡', '环境干净', '服务贴心']],
    [['健身', '瑜伽', '私教'], ['教练专业', '纠正动作', '器械齐全', '不推私教', '环境好']],
    [['ktv', '酒吧', '桌游', '剧本杀', '密室', '娱乐'], ['包厢大', '设备好', '隔音好', '服务热情', '性价比高']],
    [['洗车', '汽车', '保养', '贴膜'], ['洗得干净', '师傅仔细', '报价透明', '效率高', '环境好']],
    [['餐', '美食', '菜', '火锅', '烧烤', '饭', '小吃'], ['味道好', '招牌菜', '分量足', '适合聚餐', '服务热情', '环境舒服', '性价比高']]
  ]
  for (const [aliases, tags] of table) {
    if (aliases.some((a) => t.includes(a))) return tags
  }
  return ['服务热情', '环境舒服', '性价比高', '体验好', '干净卫生']
}

let switchCounter = 0
let remaining = 42

function pickReview(platformCode: string, tag?: string) {
  const key = reviewPool[platformCode] ? platformCode : platformCode === 'meituan' ? 'dianping' : 'dianping'
  const pool = reviewPool[key]
  const base = pool[switchCounter % pool.length]
  switchCounter += 1
  remaining = Math.max(0, remaining - 1)
  const content = base.replace(/\{\{tag\}\}/g, tag || '招牌菜')
  return {
    review: { id: nextId(), content, platformStyle: key },
    remainingDispatchableCount: remaining
  }
}

function landingPayload() {
  switchCounter = 0
  remaining = 42
  return {
    sessionId: 'mock-session-001',
    storeName: store.storeName,
    primaryPlatformStyle: store.primaryPlatformStyle,
    review: null,
    keywords,
    images,
    platformLinks,
    remainingDispatchableCount: remaining
  }
}

// ---------------- 路由表 ----------------
type Handler = (m: RegExpMatchArray, body: any) => unknown
const routes: Array<{ method: string; re: RegExp; handler: Handler }> = [
  // ----- 消费者落地页（重点）-----
  { method: 'GET', re: /\/public\/landing\/[^/]+\/init$/, handler: () => landingPayload() },
  { method: 'POST', re: /\/public\/landing\/[^/]+\/switch-review$/, handler: (_m, b) => pickReview(b.platformCode, b.tag) },
  { method: 'POST', re: /\/public\/landing\/[^/]+\/events$/, handler: () => ({ saved: true }) },

  // ----- 商家端 -----
  { method: 'POST', re: /\/merchant\/auth\/login$/, handler: () => ({ token: 'mock-merchant-token' }) },
  { method: 'GET', re: /\/merchant\/store\/detail$/, handler: () => store },
  { method: 'PUT', re: /\/merchant\/store\/detail$/, handler: (_m, b) => Object.assign(store, b) },
  { method: 'GET', re: /\/merchant\/store\/keyword-suggestions$/, handler: () => ({ tags: suggestTagsByIndustry(store.industryType) }) },
  { method: 'GET', re: /\/merchant\/store\/keywords$/, handler: () => keywords },
  { method: 'POST', re: /\/merchant\/store\/keywords$/, handler: (_m, b) => { const it = { id: nextId(), keyword: b.keyword, sortNo: b.sortNo || 0 }; keywords.push(it); return it } },
  { method: 'DELETE', re: /\/merchant\/store\/keywords\/(\d+)$/, handler: (m) => { keywords = keywords.filter((k) => k.id !== Number(m[1])); return { deleted: true } } },
  { method: 'GET', re: /\/merchant\/store\/images$/, handler: () => images },
  { method: 'POST', re: /\/merchant\/store\/images\/upload$/, handler: (_m, b) => { const it = { id: nextId(), imageUrl: b.imageUrl, thumbnailUrl: b.thumbnailUrl || b.imageUrl, status: 1, sortNo: b.sortNo || 0 }; images.push(it); return it } },
  { method: 'POST', re: /\/merchant\/store\/images\/upload-file$/, handler: () => { const u = ph('已上传图片', '#8b5cf6'); const it = { id: nextId(), imageUrl: u, thumbnailUrl: u, status: 1, sortNo: images.length + 1 }; images.push(it); return it } },
  { method: 'DELETE', re: /\/merchant\/store\/images\/(\d+)$/, handler: (m) => { images = images.filter((i) => i.id !== Number(m[1])); return { deleted: true } } },
  { method: 'GET', re: /\/merchant\/store\/platform-links$/, handler: () => platformLinks },
  { method: 'POST', re: /\/merchant\/store\/platform-links$/, handler: (_m, b) => { const it = { id: nextId(), sortNo: 0, status: 1, backupUrl: '', ...b }; platformLinks.push(it); return it } },
  { method: 'PUT', re: /\/merchant\/store\/platform-links\/(\d+)\/status$/, handler: (m, b) => { const it = platformLinks.find((p) => p.id === Number(m[1])); if (it) it.status = b.status; return it } },
  { method: 'DELETE', re: /\/merchant\/store\/platform-links\/(\d+)$/, handler: (m) => { platformLinks = platformLinks.filter((p) => p.id !== Number(m[1])); return { deleted: true } } },
  { method: 'GET', re: /\/merchant\/reviews$/, handler: () => merchantReviews },
  { method: 'POST', re: /\/merchant\/reviews$/, handler: (_m, b) => { const it = { id: nextId(), platformStyle: b.platformCode || 'xiaohongshu', content: b.content, tags: '', sourceType: 'manual', status: b.status || 'available' }; merchantReviews.unshift(it); return it } },
  { method: 'DELETE', re: /\/merchant\/reviews\/(\d+)$/, handler: (m) => { merchantReviews = merchantReviews.filter((r) => r.id !== Number(m[1])); return { deleted: true } } },
  { method: 'POST', re: /\/merchant\/reviews\/generate$/, handler: (_m, b) => {
      const n = b.targetCount || 10
      const plat = b.platformCode || 'xiaohongshu'
      const pool = reviewPool[plat] || reviewPool.dianping
      // 生成的评价进“评论列表”，便于调试
      for (let i = 0; i < Math.min(n, 3); i++) {
        merchantReviews.unshift({ id: nextId(), platformStyle: plat, content: pool[i % pool.length].replace(/\{\{tag\}\}/g, '招牌菜'), tags: '招牌菜', sourceType: 'ai', status: 'available' })
      }
      tasks.unshift({ id: nextId(), storeId: 1, platformStyle: plat, triggerType: 'manual', targetCount: n, successCount: n, failedCount: 0, status: 'success' })
      return { generated: n }
    } },
  { method: 'GET', re: /\/merchant\/review-generation-tasks$/, handler: () => tasks },

  // ----- 管理端 -----
  { method: 'POST', re: /\/admin\/auth\/login$/, handler: () => ({ token: 'mock-admin-token' }) },
  { method: 'GET', re: /\/admin\/merchants$/, handler: () => merchants },
  { method: 'PUT', re: /\/admin\/merchants\/(\d+)\/status$/, handler: (m, b) => { const it = merchants.find((x) => x.id === Number(m[1])); if (it) it.status = b.status; return it } },
  { method: 'GET', re: /\/admin\/stores$/, handler: () => stores },
  { method: 'PUT', re: /\/admin\/stores\/(\d+)\/status$/, handler: (m, b) => { const it = stores.find((x) => x.id === Number(m[1])); if (it) it.status = b.status; return it } },
  { method: 'GET', re: /\/admin\/nfc-tags$/, handler: () => nfcTags },
  { method: 'POST', re: /\/admin\/nfc-tags$/, handler: (_m, b) => { const it = { id: nextId(), tagCode: b.tagCode || `TAG-${nextId()}`, storeId: 0, landingToken: `mock-${nextId()}`, status: 'unbound', remark: b.remark || '' }; nfcTags.push(it); return it } },
  { method: 'PUT', re: /\/admin\/nfc-tags\/(\d+)\/bind$/, handler: (m, b) => { const it = nfcTags.find((t) => t.id === Number(m[1])); if (it) { it.storeId = b.storeId; it.status = 'bound' } return it } },
  { method: 'PUT', re: /\/admin\/nfc-tags\/(\d+)\/status$/, handler: (m, b) => { const it = nfcTags.find((t) => t.id === Number(m[1])); if (it) it.status = b.status; return it } },
  { method: 'GET', re: /\/admin\/review-generation-tasks$/, handler: () => tasks },
  { method: 'GET', re: /\/admin\/stats$/, handler: () => ({ merchantCount: merchants.length, storeCount: stores.length, tagCount: nfcTags.length, taskCount: tasks.length, reviewCount: merchantReviews.length, dispatchableCount: remaining }) }
]

export const mockAdapter: AxiosAdapter = async (config) => {
  const method = (config.method || 'get').toUpperCase()
  const url = (config.url || '').split('?')[0]
  let body: any = {}
  try {
    body = config.data ? JSON.parse(config.data) : {}
  } catch {
    body = {}
  }

  await delay(160 + Math.random() * 240) // 模拟网络延迟

  for (const route of routes) {
    if (route.method !== method) continue
    const m = url.match(route.re)
    if (m) {
      return {
        data: envelope(route.handler(m, body)),
        status: 200,
        statusText: 'OK',
        headers: {},
        config
      } as AxiosResponse
    }
  }

  return Promise.reject({
    response: { status: 404, data: { code: 404, message: `Mock 未实现：${method} ${url}` } },
    config
  })
}
