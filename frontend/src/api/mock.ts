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
const landingPath = (uuid: string) => `${import.meta.env.BASE_URL}landing/${uuid}`
const analyticsDataSource = 'mock_review_display_logs'
const analyticsDataSourceLabel = 'Mock 客户端落地页事件日志'

function platformHomeAppUrl(platformCode: string): string {
  switch (platformCode.trim().toLowerCase()) {
    case 'meituan': return 'imeituan://'
    case 'dianping': return 'dianping://'
    case 'douyin': return 'snssdk1128://'
    default: return ''
  }
}

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
  { id: 1, storeId: 1, platformCode: 'meituan', platformName: '美团', buttonText: '打开美团', targetUrl: 'https://w.dianping.com/cube/evoke/meituan.html', backupUrl: '', sortNo: 1, status: 1 },
  { id: 2, storeId: 1, platformCode: 'dianping', platformName: '大众点评', buttonText: '打开大众点评店铺', targetUrl: 'https://www.dianping.com', backupUrl: '', sortNo: 2, status: 1 },
  { id: 3, storeId: 1, platformCode: 'xiaohongshu', platformName: '小红书', buttonText: '打开小红书店铺', targetUrl: 'https://www.xiaohongshu.com', backupUrl: '', sortNo: 3, status: 1 },
  { id: 4, storeId: 1, platformCode: 'douyin', platformName: '抖音', buttonText: '打开抖音店铺', targetUrl: 'https://www.douyin.com', backupUrl: '', sortNo: 4, status: 1 }
]

let merchantReviews: any[] = [
  { id: 901, platformStyle: 'xiaohongshu', content: '周五和朋友来的，椒麻鸡不错，麻香不冲。', tags: '招牌椒麻鸡', sourceType: 'ai', status: 'available' }
]

let tasks = [
  {
    id: 1,
    storeId: 1,
    platformStyle: 'xiaohongshu',
    triggerType: 'manual',
    targetCount: 10,
    generatedRawCount: 10,
    insertedRowCount: 8,
    duplicateFilteredCount: 2,
    successCount: 8,
    failedCount: 2,
    status: 'partial_failed',
    errorMessage: '',
    createdAt: new Date(Date.now() - 20 * 60 * 1000).toISOString(),
    updatedAt: new Date(Date.now() - 18 * 60 * 1000).toISOString(),
    auditLogs: [
      {
        id: 101,
        taskId: 1,
        storeId: 1,
        platformStyle: 'xiaohongshu',
        triggerType: 'manual',
        stage: 'task_completed',
        level: 'info',
        status: 'partial_failed',
        message: '生成任务完成',
        detail: '{"successCount":8,"duplicateCount":2}',
        agentEndpoint: 'http://127.0.0.1:8001',
        httpStatus: 0,
        durationMs: 28430,
        targetCount: 10,
        generatedRawCount: 10,
        insertedRowCount: 8,
        duplicateFilteredCount: 2,
        createdAt: new Date(Date.now() - 18 * 60 * 1000).toISOString()
      }
    ]
  }
]

let generationPreferences = {
  configured: true,
  focusKeywords: ['招牌椒麻鸡', '服务热情'],
  styleCodes: ['natural', 'detail_rich'],
  diversityDimensions: ['customer_identity', 'content_angle'],
  referenceReviews: ['椒麻鸡麻香挺自然，服务员会主动加水。'],
  lengthVariance: 'wide',
  updatedAt: new Date().toISOString()
}

let nfcTags = [
  { id: 1, tagCode: 'TAG-DEMO-001', storeId: 1, landingToken: 'mock-demo-001', status: 'bound', remark: '演示标签' }
]

let reviewCrawlConfigs: any[] = [
  {
    id: 1,
    storeId: 1,
    platformCode: 'meituan',
    externalShopId: '1953748828',
    enabled: true,
    baselineCompletedAt: new Date(Date.now() - 8 * 24 * 60 * 60 * 1000).toISOString(),
    lastCrawledAt: new Date().toISOString(),
    nextCrawlAt: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
    lastStatus: 'success',
    lastErrorMessage: ''
  }
]

let reviewCrawlBatches: any[] = [
  {
    id: 1,
    configId: 1,
    storeId: 1,
    platformCode: 'meituan',
    externalShopIdSnapshot: '1953748828',
    triggerType: 'scheduled',
    attemptNo: 2,
    isBaseline: false,
    windowDays: 7,
    startedAt: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
    finishedAt: new Date().toISOString(),
    status: 'success',
    rawRowCount: 36,
    insertedRowCount: 36,
    matchedReviewCount: 11,
    errorMessage: ''
  }
]

let externalReviewMatches: any[] = [
  {
    id: 1,
    batchId: 1,
    storeId: 1,
    platformCode: 'meituan',
    sourceReviewRef: 'mock-1001',
    userName: '美团用户',
    ratingRaw: '50',
    reviewTime: new Date().toISOString(),
    content: '这家店服务挺热情，团购核销也顺，整体体验不错。',
    matchedFeedbackId: 1001,
    matchedReviewItemId: 901,
    matchScore: 0.92,
    matchReason: 'character_similarity',
    matchSource: 'edited_content'
  }
]

let platformReviewFewShotIds = new Set<number>([1])
let platformReviewLibrary: any[] = [
  {
    id: 1,
    batchId: 1,
    storeId: 1,
    storeName: '巷子里的椒麻鸡（Mock 演示）',
    platformCode: 'meituan',
    sourceReviewRef: 'mock-1001',
    userName: '美团用户',
    ratingRaw: '50',
    ratingNormalized: 5,
    reviewTime: new Date().toISOString(),
    content: '这家店服务挺热情，团购核销也顺，整体体验不错。',
    isBaseline: false,
    createdAt: new Date().toISOString()
  },
  {
    id: 2,
    batchId: 1,
    storeId: 1,
    storeName: '巷子里的椒麻鸡（Mock 演示）',
    platformCode: 'meituan',
    sourceReviewRef: 'mock-1002',
    userName: '匿名用户',
    ratingRaw: '45',
    ratingNormalized: 4.5,
    reviewTime: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
    content: '椒麻鸡味道比较稳，朋友聚餐点一桌也不会太踩雷，服务员补水挺主动。',
    isBaseline: false,
    createdAt: new Date().toISOString()
  },
  {
    id: 3,
    batchId: 2,
    storeId: 2,
    storeName: '舒缘足道',
    platformCode: 'dianping',
    sourceReviewRef: 'mock-2001',
    userName: '点评用户',
    ratingRaw: '50',
    ratingNormalized: 5,
    reviewTime: new Date(Date.now() - 4 * 24 * 60 * 60 * 1000).toISOString(),
    content: '技师手法挺专业，力度会提前确认，按完肩颈轻松不少。',
    isBaseline: true,
    createdAt: new Date().toISOString()
  }
]

// 多商家：演示「每个商家有自己独立的数据」（管理员能看到全部，商家只看到自己的）
const merchants = [
  { id: 1, account: 'merchant', merchantName: '巷子里的椒麻鸡', contactName: '张三', status: 1, createdAt: new Date(Date.now() - 9 * 24 * 60 * 60 * 1000).toISOString() },
  { id: 2, account: 'merchant2', merchantName: '舒缘足道', contactName: '李四', status: 1, createdAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString() }
]
const storeTypes = [
  { id: 1, code: 'restaurant', name: '餐饮', industryCode: 'restaurant', isPreset: true, status: 1 },
  { id: 2, code: 'footmassage', name: '足疗按摩', industryCode: 'footmassage', isPreset: true, status: 1 },
  { id: 3, code: 'hairsalon', name: '理发美发', industryCode: 'hairsalon', isPreset: true, status: 1 },
  { id: 4, code: 'nailsalon', name: '美甲美睫', industryCode: 'nailsalon', isPreset: true, status: 1 },
  { id: 5, code: 'beauty', name: '美容护肤', industryCode: 'beauty', isPreset: true, status: 1 },
  { id: 6, code: 'fitness', name: '健身运动', industryCode: 'fitness', isPreset: true, status: 1 },
  { id: 7, code: 'entertainment', name: '休闲娱乐', industryCode: 'entertainment', isPreset: true, status: 1 },
  { id: 8, code: 'pet', name: '宠物服务', industryCode: 'pet', isPreset: true, status: 1 },
  { id: 9, code: 'auto', name: '汽车服务', industryCode: 'auto', isPreset: true, status: 1 }
]
const stores: any[] = [
  { ...store, id: 1, merchantUserId: 1, createdAt: new Date(Date.now() - 9 * 24 * 60 * 60 * 1000).toISOString() },
  { id: 2, merchantUserId: 2, uuid: '22222222-2222-4222-8222-222222222222', typeId: 2, storeName: '舒缘足道', industryType: '足疗按摩', storeIntro: '', address: '', primaryPlatformStyle: 'dianping', brandTone: '轻松自然', status: 1, createdAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString() }
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
    review: { id: nextId(), content, platformStyle: platformCode === 'meituan' ? 'meituan' : key },
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
    platformLinks: platformLinks.map((link) => ({
      ...link,
      openMode: platformHomeAppUrl(link.platformCode) ? 'app_link' : 'official_link',
      openUrl: platformHomeAppUrl(link.platformCode) || link.targetUrl || link.backupUrl || ''
    })),
    remainingDispatchableCount: remaining
  }
}

function deleteStoreById(id: number) {
  const index = stores.findIndex((s) => s.id === id)
  if (index < 0) return { deleted: false }
  stores.splice(index, 1)
  platformLinks = platformLinks.filter((link) => link.storeId !== id)
  nfcTags = nfcTags.map((tag) =>
    tag.storeId === id ? { ...tag, storeId: 0, landingToken: '', status: 'unbound' } : tag
  )
  tasks = tasks.filter((task) => task.storeId !== id)
  const configIds = reviewCrawlConfigs.filter((cfg) => cfg.storeId === id).map((cfg) => cfg.id)
  reviewCrawlConfigs = reviewCrawlConfigs.filter((cfg) => cfg.storeId !== id)
  reviewCrawlBatches = reviewCrawlBatches.filter((batch) => !configIds.includes(batch.configId))
  externalReviewMatches = externalReviewMatches.filter((match) => match.storeId !== id)
  if (id === store.id) {
    keywords = []
    images = []
    merchantReviews = []
  }
  return { deleted: true }
}

function deleteMerchantById(id: number) {
  const merchantIndex = merchants.findIndex((m) => m.id === id)
  if (merchantIndex < 0) return { deleted: false }
  const storeIds = stores.filter((s) => s.merchantUserId === id).map((s) => s.id)
  for (const storeId of storeIds) {
    deleteStoreById(storeId)
  }
  merchants.splice(merchantIndex, 1)
  return { deleted: true }
}

const platformStatSeeds: Record<string, { weekly: number[]; monthly: number[]; weeklyVisits: number[]; monthlyVisits: number[]; weeklyShare: number; monthlyShare: number }> = {
  meituan: {
    weekly: [3, 5, 0, 8, 12, 9, 14, 18, 15, 21, 17, 24],
    monthly: [22, 28, 31, 35, 42, 40, 46, 51, 48, 56, 61, 66],
    weeklyVisits: [19, 24, 18, 35, 45, 40, 52, 68, 59, 77, 83, 96],
    monthlyVisits: [88, 102, 128, 141, 166, 158, 181, 204, 221, 246, 270, 312],
    weeklyShare: 42.9,
    monthlyShare: 31.4
  },
  dianping: {
    weekly: [2, 3, 4, 6, 7, 8, 9, 13, 12, 16, 18, 21],
    monthly: [14, 17, 19, 22, 26, 29, 33, 34, 39, 43, 48, 52],
    weeklyVisits: [14, 16, 18, 21, 24, 31, 35, 42, 44, 51, 57, 63],
    monthlyVisits: [61, 72, 80, 92, 104, 118, 132, 146, 161, 179, 193, 216],
    weeklyShare: 36.2,
    monthlyShare: 28.8
  },
  xiaohongshu: {
    weekly: [1, 2, 3, 5, 6, 8, 11, 12, 14, 13, 15, 19],
    monthly: [9, 12, 15, 19, 22, 26, 30, 35, 37, 42, 44, 49],
    weeklyVisits: [10, 12, 15, 18, 22, 26, 34, 39, 43, 41, 48, 55],
    monthlyVisits: [46, 55, 68, 76, 91, 106, 119, 138, 151, 166, 181, 204],
    weeklyShare: 39.6,
    monthlyShare: 33.1
  },
  douyin: {
    weekly: [1, 1, 2, 2, 5, 4, 6, 9, 8, 11, 10, 8],
    monthly: [7, 9, 13, 16, 18, 23, 25, 29, 31, 36, 39, 34],
    weeklyVisits: [8, 9, 12, 13, 18, 17, 23, 29, 31, 35, 37, 34],
    monthlyVisits: [34, 41, 49, 58, 67, 80, 91, 104, 118, 130, 147, 139],
    weeklyShare: 22.4,
    monthlyShare: 24.5
  }
}

function sumSeries(series: number[][]) {
  const length = Math.max(...series.map((items) => items.length))
  return Array.from({ length }, (_, index) => series.reduce((sum, items) => sum + (items[index] || 0), 0))
}

function mockPublishStats(params: Record<string, unknown> = {}) {
  const requestedPlatform = String(params.platformCode || '').trim()
  const selectedSeed = requestedPlatform ? platformStatSeeds[requestedPlatform] : null
  const activeSeeds = Object.values(platformStatSeeds)
  const weeklyCounts = selectedSeed?.weekly || sumSeries(activeSeeds.map((item) => item.weekly))
  const monthlyCounts = selectedSeed?.monthly || sumSeries(activeSeeds.map((item) => item.monthly))
  const weeklyVisitCounts = selectedSeed?.weeklyVisits || sumSeries(activeSeeds.map((item) => item.weeklyVisits))
  const monthlyVisitCounts = selectedSeed?.monthlyVisits || sumSeries(activeSeeds.map((item) => item.monthlyVisits))
  const weeklyShare = selectedSeed?.weeklyShare || 35.8
  const monthlyShare = selectedSeed?.monthlyShare || 29.7
  const today = new Date()
  const weeklySeries = weeklyCounts.map((count, index) => {
    const start = new Date(today)
    start.setDate(today.getDate() - (weeklyCounts.length - 1 - index) * 7 - today.getDay() + 1)
    const end = new Date(start)
    end.setDate(start.getDate() + 6)
    return { weekStart: start.toISOString().slice(0, 10), weekEnd: end.toISOString().slice(0, 10), count }
  })
  const monthlySeries = monthlyCounts.map((count, index) => {
    const month = new Date(today.getFullYear(), today.getMonth() - (monthlyCounts.length - 1 - index), 1)
    return { month: month.toISOString().slice(0, 7), count }
  })
  return {
    platformCode: selectedSeed ? requestedPlatform : '',
    platformName: selectedSeed ? platformName(requestedPlatform) : '全部平台',
    totalPublishClicks: weeklyCounts.reduce((sum, count) => sum + count, 0),
    currentWeekPublishClicks: weeklyCounts[weeklyCounts.length - 1],
    currentMonthPublishClicks: monthlyCounts[monthlyCounts.length - 1],
    previousWeekPublishClicks: weeklyCounts[weeklyCounts.length - 2],
    previousMonthPublishClicks: monthlyCounts[monthlyCounts.length - 2],
    publishWeekGrowthPercent: growthPercent(weeklyCounts[weeklyCounts.length - 1], weeklyCounts[weeklyCounts.length - 2]),
    publishMonthGrowthPercent: growthPercent(monthlyCounts[monthlyCounts.length - 1], monthlyCounts[monthlyCounts.length - 2]),
    totalCustomerVisits: weeklyVisitCounts.reduce((sum, count) => sum + count, 0),
    currentWeekCustomerVisits: weeklyVisitCounts[weeklyVisitCounts.length - 1],
    currentMonthCustomerVisits: monthlyVisitCounts[monthlyVisitCounts.length - 1],
    previousWeekCustomerVisits: weeklyVisitCounts[weeklyVisitCounts.length - 2],
    previousMonthCustomerVisits: monthlyVisitCounts[monthlyVisitCounts.length - 2],
    visitWeekGrowthPercent: growthPercent(weeklyVisitCounts[weeklyVisitCounts.length - 1], weeklyVisitCounts[weeklyVisitCounts.length - 2]),
    visitMonthGrowthPercent: growthPercent(monthlyVisitCounts[monthlyVisitCounts.length - 1], monthlyVisitCounts[monthlyVisitCounts.length - 2]),
    updatedAt: new Date().toISOString(),
    dataSource: analyticsDataSource,
    dataSourceLabel: analyticsDataSourceLabel,
    timezone: 'Asia/Shanghai',
    currentWeekStart: weeklySeries[weeklySeries.length - 1].weekStart,
    currentWeekEnd: weeklySeries[weeklySeries.length - 1].weekEnd,
    currentMonthStart: `${monthlySeries[monthlySeries.length - 1].month}-01`,
    currentMonthEnd: `${monthlySeries[monthlySeries.length - 1].month}-30`,
    platformLinksConfigured: platformLinks.some((link) => link.status === 1),
    activePlatformLinkCount: platformLinks.filter((link) => link.status === 1).length,
    crawlDataReady: true,
    crawlDataMessage: '',
    weeklyGuidedShareReady: true,
    monthlyGuidedShareReady: true,
    weeklyGuidedSharePercent: weeklyShare,
    monthlyGuidedSharePercent: monthlyShare,
    deviceStats: mockDeviceStats(weeklyVisitCounts.reduce((sum, count) => sum + count, 0)),
    weeklySeries,
    monthlySeries,
    partialErrors: []
  }
}

function mockDeviceStats(total: number) {
  const safeTotal = Math.max(total, 0)
  if (safeTotal === 0) return { totalCount: 0, items: [] }
  const iphone = Math.round(safeTotal * 0.43)
  const huawei = Math.round(safeTotal * 0.18)
  const xiaomi = Math.round(safeTotal * 0.14)
  const oppo = Math.round(safeTotal * 0.11)
  const vivo = Math.round(safeTotal * 0.08)
  const androidOther = Math.max(0, safeTotal - iphone - huawei - xiaomi - oppo - vivo)
  return {
    totalCount: safeTotal,
    items: [
      { code: 'iphone', label: '苹果 iPhone', count: iphone, percent: Number(((iphone / safeTotal) * 100).toFixed(1)) },
      { code: 'huawei', label: '华为', count: huawei, percent: Number(((huawei / safeTotal) * 100).toFixed(1)) },
      { code: 'xiaomi', label: '小米/Redmi', count: xiaomi, percent: Number(((xiaomi / safeTotal) * 100).toFixed(1)) },
      { code: 'oppo', label: 'OPPO/一加/realme', count: oppo, percent: Number(((oppo / safeTotal) * 100).toFixed(1)) },
      { code: 'vivo', label: 'vivo/iQOO', count: vivo, percent: Number(((vivo / safeTotal) * 100).toFixed(1)) },
      { code: 'android_other', label: 'Android 其他', count: androidOther, percent: Number(((androidOther / safeTotal) * 100).toFixed(1)) }
    ].filter((item) => item.count > 0)
  }
}

function growthPercent(current: number, previous: number) {
  if (!previous) return current > 0 ? 100 : 0
  return Number((((current - previous) / previous) * 100).toFixed(1))
}

function mockStoreAnalytics(id: number) {
  const visits = id === 1 ? 684 : 238
  const publishes = id === 1 ? 146 : 41
  return {
    totalCustomerVisits: visits,
    currentWeekCustomerVisits: id === 1 ? 96 : 28,
    currentMonthCustomerVisits: id === 1 ? 312 : 93,
    totalPublishClicks: publishes,
    currentWeekPublishClicks: id === 1 ? 24 : 9,
    currentMonthPublishClicks: id === 1 ? 66 : 18,
    activePlatformLinkCount: platformLinks.filter((link) => link.storeId === id && link.status === 1).length,
    deviceStats: mockDeviceStats(visits),
    dataSource: analyticsDataSource,
    dataSourceLabel: analyticsDataSourceLabel
  }
}

function platformName(code: string) {
  return ({ dianping: '大众点评', meituan: '美团', xiaohongshu: '小红书', douyin: '抖音' } as Record<string, string>)[code] || code
}

function adminStoreView(item: any) {
  const merchant = merchants.find((m) => m.id === item.merchantUserId)
  const link = platformLinks.find((p) => p.storeId === item.id && p.platformCode === item.primaryPlatformStyle)
  const reviewCrawl = reviewCrawlConfigs.find((cfg) => cfg.storeId === item.id)
  const storeTags = nfcTags.filter((tag) => tag.storeId === item.id)
  const writtenCount = storeTags.filter((tag) => tag.status === 'bound').length
  const disabledCount = storeTags.filter((tag) => tag.status === 'disabled').length
  const primaryStatus = !item.uuid
    ? 'unwritten'
    : item.status !== 1
      ? 'unusable'
      : writtenCount > 0
        ? 'usable'
        : 'unwritten'
  return {
    ...item,
    merchantAccount: merchant?.account || '',
    merchantName: merchant?.merchantName || '',
    contactName: merchant?.contactName || '',
    platformUrl: link?.targetUrl || '',
    landingUrl: landingPath(item.uuid),
    analytics: mockStoreAnalytics(item.id),
    nfcCardStatus: {
      totalCount: storeTags.length,
      writtenCount,
      disabledCount,
      primaryStatus,
      routeStatus: primaryStatus === 'usable' ? 'ok' : writtenCount > 0 ? 'store_inactive' : 'no_bound_tag'
    },
    reviewCrawl
  }
}

function saveMockPlatformLink(storeId: number, platformCode: string, targetUrl?: string) {
  const index = platformLinks.findIndex((p) => p.storeId === storeId && p.platformCode === platformCode)
  const value = String(targetUrl || '').trim()
  if (!value) {
    if (index >= 0) platformLinks.splice(index, 1)
    return
  }
  const next = {
    id: index >= 0 ? platformLinks[index].id : nextId(),
    storeId,
    platformCode,
    platformName: platformName(platformCode),
    buttonText: `去${platformName(platformCode)}发布`,
    targetUrl: value,
    backupUrl: value,
    sortNo: 1,
    status: 1
  }
  if (index >= 0) platformLinks[index] = next
  else platformLinks.push(next)
}

function saveMockReviewCrawlConfig(storeId: number, body: any) {
  const enabled = !!body.reviewCrawlEnabled
  const externalShopId = String(body.reviewCrawlExternalShopId || '').trim()
  const platformCode = String(body.reviewCrawlPlatformCode || 'meituan').trim() || 'meituan'
  const index = reviewCrawlConfigs.findIndex((cfg) => cfg.storeId === storeId)
  if (!enabled && !externalShopId && !body.reviewCrawlPlatformCode) {
    if (index >= 0) reviewCrawlConfigs[index] = { ...reviewCrawlConfigs[index], enabled: false, externalShopId: '', lastStatus: reviewCrawlConfigs[index].lastStatus || 'never_run' }
    return
  }
  const next = {
    id: index >= 0 ? reviewCrawlConfigs[index].id : nextId(),
    storeId,
    platformCode,
    externalShopId,
    enabled: enabled && !!externalShopId,
    baselineCompletedAt: index >= 0 ? reviewCrawlConfigs[index].baselineCompletedAt : '',
    lastCrawledAt: index >= 0 ? reviewCrawlConfigs[index].lastCrawledAt : '',
    nextCrawlAt: index >= 0 ? reviewCrawlConfigs[index].nextCrawlAt : '',
    lastStatus: index >= 0 ? reviewCrawlConfigs[index].lastStatus : 'never_run',
    lastErrorMessage: ''
  }
  if (index >= 0) reviewCrawlConfigs[index] = next
  else reviewCrawlConfigs.push(next)
}

// ---------------- 路由表 ----------------
type Handler = (m: RegExpMatchArray, body: any, params: Record<string, unknown>) => unknown

function mockFailure(status: number, message: string): never {
  const err = new Error(message) as Error & { status?: number }
  err.status = status
  throw err
}

function mockPlatformReviewLibrary(params: Record<string, unknown>) {
  const storeId = Number(params.storeId || 0)
  const platformCode = String(params.platformCode || '')
  const q = String(params.q || '').trim().toLowerCase()
  const selectedOnly = params.selectedOnly === true || params.selectedOnly === 'true'
  const limit = Math.max(1, Math.min(Number(params.limit || 80), 200))
  const offset = Math.max(0, Number(params.offset || 0))
  let items = platformReviewLibrary.map((item) => ({
    ...item,
    isFewShot: platformReviewFewShotIds.has(item.id),
    selectedAt: platformReviewFewShotIds.has(item.id) ? new Date().toISOString() : undefined
  }))
  if (storeId) items = items.filter((item) => item.storeId === storeId)
  if (platformCode) items = items.filter((item) => item.platformCode === platformCode)
  if (selectedOnly) items = items.filter((item) => item.isFewShot)
  if (q) {
    items = items.filter((item) =>
      [item.storeName, item.userName, item.content, item.sourceReviewRef].some((value) =>
        String(value || '').toLowerCase().includes(q)
      )
    )
  }
  const total = items.length
  const selectedCount = items.filter((item) => item.isFewShot).length
  return { items: items.slice(offset, offset + limit), total, selectedCount, limit, offset }
}

const routes: Array<{ method: string; re: RegExp; handler: Handler }> = [
  // ----- 消费者落地页（重点）-----
  { method: 'GET', re: /\/public\/landing\/[^/]+\/init$/, handler: () => landingPayload() },
  { method: 'POST', re: /\/public\/landing\/[^/]+\/switch-review$/, handler: (_m, b) => pickReview(b.platformCode, b.tag) },
  { method: 'POST', re: /\/public\/landing\/[^/]+\/events$/, handler: () => ({ saved: true }) },

  // ----- 商家端 -----
  { method: 'POST', re: /\/merchant\/auth\/login$/, handler: () => ({ token: 'mock-merchant-token' }) },
  { method: 'GET', re: /\/merchant\/store\/detail$/, handler: () => store },
  { method: 'PUT', re: /\/merchant\/store\/detail$/, handler: (_m, b) => Object.assign(store, b) },
  { method: 'GET', re: /\/merchant\/dashboard\/publish-stats$/, handler: (_m, _b, params) => mockPublishStats(params) },
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
  { method: 'PUT', re: /\/merchant\/store\/platform-links\/(\d+)$/, handler: (m, b) => { const it = platformLinks.find((p) => p.id === Number(m[1])); if (it) Object.assign(it, b); return it } },
  { method: 'PUT', re: /\/merchant\/store\/platform-links\/(\d+)\/status$/, handler: (m, b) => { const it = platformLinks.find((p) => p.id === Number(m[1])); if (it) it.status = b.status; return it } },
  { method: 'DELETE', re: /\/merchant\/store\/platform-links\/(\d+)$/, handler: (m) => { platformLinks = platformLinks.filter((p) => p.id !== Number(m[1])); return { deleted: true } } },
  { method: 'GET', re: /\/merchant\/reviews$/, handler: () => merchantReviews },
  { method: 'POST', re: /\/merchant\/reviews$/, handler: (_m, b) => { const it = { id: nextId(), platformStyle: b.platformCode || 'xiaohongshu', content: b.content, tags: '', sourceType: 'manual', status: b.status || 'available' }; merchantReviews.unshift(it); return it } },
  { method: 'DELETE', re: /\/merchant\/reviews\/(\d+)$/, handler: (m) => { merchantReviews = merchantReviews.filter((r) => r.id !== Number(m[1])); return { deleted: true } } },
  { method: 'GET', re: /\/merchant\/review-generation-preferences$/, handler: () => generationPreferences },
  { method: 'PUT', re: /\/merchant\/review-generation-preferences$/, handler: (_m, b) => { generationPreferences = { configured: true, focusKeywords: b.focusKeywords || [], styleCodes: b.styleCodes || ['natural'], diversityDimensions: b.diversityDimensions || ['customer_identity'], referenceReviews: b.referenceReviews || [], lengthVariance: b.lengthVariance || 'wide', updatedAt: new Date().toISOString() }; return generationPreferences } },
  { method: 'POST', re: /\/merchant\/reviews\/generate$/, handler: () => mockFailure(503, 'Mock 模式不生成评价，请连接真实后端和 agent-service') },
  { method: 'GET', re: /\/merchant\/review-generation-tasks$/, handler: () => tasks },

  // ----- 管理端 -----
  { method: 'POST', re: /\/admin\/auth\/login$/, handler: () => ({ token: 'mock-admin-token' }) },
  { method: 'GET', re: /\/admin\/merchants$/, handler: () => merchants },
  { method: 'PUT', re: /\/admin\/merchants\/(\d+)\/status$/, handler: (m, b) => { const it = merchants.find((x) => x.id === Number(m[1])); if (it) it.status = b.status; return it } },
  { method: 'DELETE', re: /\/admin\/merchants\/(\d+)$/, handler: (m) => deleteMerchantById(Number(m[1])) },
  { method: 'GET', re: /\/admin\/store-types$/, handler: () => storeTypes },
  { method: 'POST', re: /\/admin\/store-types$/, handler: (_m, b) => { const it = { id: nextId(), code: `custom-${nextId()}`, name: b.name, industryCode: b.industryCode, isPreset: false, status: 1 }; storeTypes.push(it); return it } },
  { method: 'GET', re: /\/admin\/stores$/, handler: () => stores.map(adminStoreView) },
  { method: 'POST', re: /\/admin\/stores$/, handler: (_m, b) => {
    const t = storeTypes.find((x) => x.id === Number(b.typeId))
    const mid = nextId(); const sid = nextId(); const uuid = `mock-${sid}-${Date.now()}`
    const createdAt = new Date().toISOString()
    merchants.push({ id: mid, account: b.account, merchantName: b.merchantName || b.storeName, contactName: b.contactName || b.merchantName || b.storeName, status: 1, createdAt })
    const s = { id: sid, merchantUserId: mid, uuid, typeId: t?.id || 0, storeName: b.storeName, industryType: t?.name || '餐饮', storeIntro: b.storeIntro || '', address: b.address || '', primaryPlatformStyle: b.primaryPlatformStyle || 'dianping', brandTone: b.brandTone || '轻松自然', status: 1, createdAt }
    stores.push(s)
    saveMockPlatformLink(sid, s.primaryPlatformStyle, b.platformUrl)
    saveMockReviewCrawlConfig(sid, b)
    return { store: s, merchant: { id: mid, account: b.account }, landingUrl: landingPath(uuid) }
  } },
  { method: 'PUT', re: /\/admin\/stores\/(\d+)$/, handler: (m, b) => {
    const item = stores.find((x) => x.id === Number(m[1]))
    if (!item) return null
    const t = storeTypes.find((x) => x.id === Number(b.typeId))
    const merchant = merchants.find((x) => x.id === item.merchantUserId)
    if (merchant) {
      merchant.account = b.account
      merchant.merchantName = b.merchantName || b.storeName
      merchant.contactName = b.contactName || merchant.merchantName
    }
    Object.assign(item, {
      typeId: t?.id || item.typeId,
      storeName: b.storeName,
      industryType: t?.name || item.industryType,
      storeIntro: b.storeIntro || '',
      address: b.address || '',
      primaryPlatformStyle: b.primaryPlatformStyle || 'dianping',
      brandTone: b.brandTone || ''
    })
    saveMockPlatformLink(item.id, item.primaryPlatformStyle, b.platformUrl)
    saveMockReviewCrawlConfig(item.id, b)
    return adminStoreView(item)
  } },
  { method: 'POST', re: /\/admin\/stores\/(\d+)\/reviews\/regenerate$/, handler: (m, b) => {
    const storeId = Number(m[1])
    const item = stores.find((x) => x.id === storeId)
    if (!item) mockFailure(404, '门店不存在')
    const platformCode = b.platformCode || item.primaryPlatformStyle || 'dianping'
    const cleared = merchantReviews.filter((review: any) =>
      (!review.storeId || review.storeId === storeId) &&
      review.platformStyle === platformCode &&
      review.status !== 'deleted'
    ).length
    merchantReviews = merchantReviews.filter((review: any) =>
      !((!review.storeId || review.storeId === storeId) && review.platformStyle === platformCode)
    )
    const targetCount = Number(b.targetCount) || 10
    for (let i = 0; i < Math.min(targetCount, 3); i += 1) {
      merchantReviews.unshift({
        id: nextId(),
        storeId,
        platformStyle: platformCode,
        content: `重新生成的评价 ${i + 1}：上周和朋友过去，体验挺自然，服务也主动。`,
        tags: '重新生成',
        sourceType: 'ai',
        status: 'available'
      })
    }
    const now = new Date().toISOString()
    const taskId = nextId()
    tasks.unshift({
      id: taskId,
      storeId,
      platformStyle: platformCode,
      triggerType: 'admin_regenerate',
      targetCount,
      generatedRawCount: targetCount,
      insertedRowCount: targetCount,
      duplicateFilteredCount: 0,
      successCount: targetCount,
      failedCount: 0,
      status: 'success',
      errorMessage: '',
      createdAt: now,
      updatedAt: now,
      auditLogs: []
    })
    return { cleared, generated: targetCount, platformCode }
  } },
  { method: 'POST', re: /\/admin\/stores\/(\d+)\/review-crawl\/run$/, handler: (m) => {
    const storeId = Number(m[1])
    const cfg = reviewCrawlConfigs.find((item) => item.storeId === storeId)
    if (!cfg || !cfg.enabled) mockFailure(400, '门店未启用评论采集')
    const now = new Date()
    const batch = {
      id: nextId(),
      configId: cfg.id,
      storeId,
      platformCode: cfg.platformCode,
      externalShopIdSnapshot: cfg.externalShopId,
      triggerType: 'manual',
      attemptNo: reviewCrawlBatches.filter((item) => item.configId === cfg.id).length + 1,
      isBaseline: !cfg.baselineCompletedAt,
      windowDays: 7,
      startedAt: new Date(now.getTime() - 2 * 60 * 1000).toISOString(),
      finishedAt: now.toISOString(),
      status: 'success',
      rawRowCount: 36,
      insertedRowCount: 36,
      matchedReviewCount: cfg.baselineCompletedAt ? 11 : 0,
      errorMessage: ''
    }
    reviewCrawlBatches.unshift(batch)
    cfg.lastStatus = 'success'
    cfg.lastErrorMessage = ''
    cfg.lastCrawledAt = now.toISOString()
    cfg.nextCrawlAt = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000).toISOString()
    if (!cfg.baselineCompletedAt) cfg.baselineCompletedAt = now.toISOString()
    return { batch, skipped: false, message: '采集完成' }
  } },
  { method: 'GET', re: /\/admin\/stores\/(\d+)\/review-crawl\/batches$/, handler: (m) => {
    const storeId = Number(m[1])
    return reviewCrawlBatches.filter((item) => item.storeId === storeId)
  } },
  { method: 'GET', re: /\/admin\/stores\/(\d+)\/review-crawl\/matches$/, handler: (m) => {
    const storeId = Number(m[1])
    return externalReviewMatches.filter((item) => item.storeId === storeId)
  } },
  { method: 'GET', re: /\/admin\/platform-reviews$/, handler: (_m, _b, params) => mockPlatformReviewLibrary(params) },
  { method: 'PUT', re: /\/admin\/platform-reviews\/(\d+)\/few-shot$/, handler: (m, b) => {
    const id = Number(m[1])
    if (!platformReviewLibrary.some((item) => item.id === id)) mockFailure(404, '平台评论不存在')
    if (b.selected) platformReviewFewShotIds.add(id)
    else platformReviewFewShotIds.delete(id)
    return { id, selected: !!b.selected }
  } },
  { method: 'PUT', re: /\/admin\/stores\/(\d+)\/status$/, handler: (m, b) => { const it = stores.find((x) => x.id === Number(m[1])); if (it) it.status = b.status; return it } },
  { method: 'DELETE', re: /\/admin\/stores\/(\d+)$/, handler: (m) => {
    const storeId = Number(m[1])
    const target = stores.find((s) => s.id === storeId)
    const result = deleteStoreById(storeId)
    if (target) {
      const merchantIndex = merchants.findIndex((merchant) => merchant.id === target.merchantUserId)
      if (merchantIndex >= 0) merchants.splice(merchantIndex, 1)
    }
    return result
  } },
  { method: 'GET', re: /\/admin\/nfc-tags$/, handler: () => nfcTags },
  { method: 'POST', re: /\/admin\/nfc-tags$/, handler: (_m, b) => { const sid = Number(b.storeId) || 0; const it = { id: nextId(), tagCode: b.tagCode || `TAG-${nextId()}`, storeId: sid, landingToken: '', status: sid ? 'bound' : 'unbound', remark: b.remark || '' }; nfcTags.push(it); return it } },
  { method: 'PUT', re: /\/admin\/nfc-tags\/(\d+)\/bind$/, handler: (m, b) => { const it = nfcTags.find((t) => t.id === Number(m[1])); if (it) { it.storeId = b.storeId; it.status = 'bound' } return it } },
  { method: 'PUT', re: /\/admin\/nfc-tags\/(\d+)\/status$/, handler: (m, b) => { const it = nfcTags.find((t) => t.id === Number(m[1])); if (it) it.status = b.status; return it } },
  { method: 'GET', re: /\/admin\/review-generation-tasks$/, handler: () => tasks },
  { method: 'GET', re: /\/admin\/stats$/, handler: () => {
    const totalVisits = stores.reduce((sum, item) => sum + mockStoreAnalytics(item.id).totalCustomerVisits, 0)
    const totalPublishClicks = stores.reduce((sum, item) => sum + mockStoreAnalytics(item.id).totalPublishClicks, 0)
    const now = new Date()
    const weekStart = new Date(now)
    weekStart.setDate(now.getDate() - (now.getDay() || 7) + 1)
    weekStart.setHours(0, 0, 0, 0)
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1)
    const createdAfter = (date: Date) => merchants.filter((item: any) => new Date(item.createdAt || 0) >= date).length
    return {
      merchantCount: merchants.length,
      enabledMerchantCount: merchants.filter((item) => item.status === 1).length,
      disabledMerchantCount: merchants.filter((item) => item.status !== 1).length,
      currentWeekNewMerchants: createdAfter(weekStart),
      currentMonthNewMerchants: createdAfter(monthStart),
      storeCount: stores.length,
      enabledStoreCount: stores.filter((item) => item.status === 1).length,
      disabledStoreCount: stores.filter((item) => item.status !== 1).length,
      tagCount: nfcTags.length,
      taskCount: tasks.length,
      crawlEnabledStoreCount: reviewCrawlConfigs.filter((item) => item.enabled).length,
      crawlFailedStoreCount: reviewCrawlConfigs.filter((item) => item.enabled && item.lastStatus === 'failed').length,
      crawlDataAccumulatingCount: reviewCrawlConfigs.filter((item) => item.enabled && !item.baselineCompletedAt).length,
      totalCustomerVisits: totalVisits,
      currentWeekCustomerVisits: stores.reduce((sum, item) => sum + mockStoreAnalytics(item.id).currentWeekCustomerVisits, 0),
      currentMonthCustomerVisits: stores.reduce((sum, item) => sum + mockStoreAnalytics(item.id).currentMonthCustomerVisits, 0),
      totalPublishClicks,
      currentWeekPublishClicks: stores.reduce((sum, item) => sum + mockStoreAnalytics(item.id).currentWeekPublishClicks, 0),
      currentMonthPublishClicks: stores.reduce((sum, item) => sum + mockStoreAnalytics(item.id).currentMonthPublishClicks, 0),
      deviceStats: mockDeviceStats(totalVisits),
      dataSource: analyticsDataSource,
      dataSourceLabel: analyticsDataSourceLabel,
      updatedAt: new Date().toISOString()
    }
  } }
]

export const mockAdapter: AxiosAdapter = async (config) => {
  const method = (config.method || 'get').toUpperCase()
  const url = (config.url || '').split('?')[0]
  const params = (config.params || {}) as Record<string, unknown>
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
      try {
        return {
          data: envelope(route.handler(m, body, params)),
          status: 200,
          statusText: 'OK',
          headers: {},
          config
        } as AxiosResponse
      } catch (err: any) {
        const status = err?.status || 500
        return Promise.reject({
          response: { status, data: { code: status, message: err?.message || 'Mock 请求失败' } },
          config
        })
      }
    }
  }

  return Promise.reject({
    response: { status: 404, data: { code: 404, message: `Mock 未实现：${method} ${url}` } },
    config
  })
}
