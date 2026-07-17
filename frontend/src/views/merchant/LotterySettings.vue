<script setup lang="ts">
import { reactive, watch } from 'vue'
import type { LotteryConfig } from '../../api/merchant'

const props = defineProps<{ config: LotteryConfig; saving: boolean }>()
const emit = defineEmits<{ save: [config: LotteryConfig] }>()

const form = reactive<LotteryConfig>({ enabled: false, prizes: [] })

function copyConfig(config: LotteryConfig) {
  form.enabled = config.enabled
  form.prizes = config.prizes.map((item) => ({ ...item }))
}

watch(() => props.config, copyConfig, { immediate: true, deep: true })

function addPrize() {
  if (form.prizes.length >= 12) return
  form.prizes.push({ name: '', imageUrl: '', stock: 10, winRate: 10, enabled: true })
}

function removePrize(index: number) {
  form.prizes.splice(index, 1)
}

function save() {
  emit('save', { enabled: form.enabled, prizes: form.prizes.map((item) => ({ ...item })) })
}
</script>

<template>
  <details class="card lottery-card" data-effect-target="lottery">
    <summary class="lottery-header">
      <span>
        <strong id="lottery-settings-title">到店互动抽奖</strong>
        <small>{{ form.enabled ? '已开启：顾客返回页面后即时出结果' : '未开启：顾客不会看到抽奖入口' }}</small>
      </span>
      <span class="lottery-summary-action">配置奖品 <span class="lottery-chevron" aria-hidden="true">⌄</span></span>
    </summary>
    <div class="lottery-content" aria-labelledby="lottery-settings-title">
      <label class="lottery-switch">
        <input v-model="form.enabled" data-testid="lottery-enabled" type="checkbox" />
        <span class="lottery-toggle" aria-hidden="true"></span>
        <span>开启抽奖活动</span>
      </label>
      <p class="lottery-help">顾客完成“复制并打开店铺”后，返回本页即可抽取奖品；中奖时由店员当场赠送，无需支付、会员或后续核销。</p>

      <div class="lottery-prize-list">
        <article v-for="(prize, index) in form.prizes" :key="prize.id || `new-${index}`" class="lottery-prize-row">
          <div class="lottery-prize-head">
            <strong>奖品 {{ index + 1 }}</strong>
            <label class="prize-enabled"><input v-model="prize.enabled" type="checkbox" /><span>{{ prize.enabled ? '✓ 已启用' : '未启用' }}</span></label>
            <button type="button" class="danger" @click="removePrize(index)">删除</button>
          </div>
          <div class="lottery-prize-fields">
            <label>名称<input v-model.trim="prize.name" data-testid="lottery-prize-name" maxlength="64" placeholder="如：招牌小吃一份" /></label>
            <label>库存<input v-model.number="prize.stock" type="number" min="0" max="100000" /></label>
            <label>中奖概率<input v-model.number="prize.winRate" type="number" min="0" max="100" /><span>%</span></label>
            <label class="lottery-image">图片地址（选填）<input v-model.trim="prize.imageUrl" placeholder="https://…" /></label>
          </div>
        </article>
      </div>
      <div class="row action-row">
        <button type="button" class="secondary" data-testid="add-lottery-prize" :disabled="form.prizes.length >= 12 || saving" @click="addPrize">新增奖品</button>
        <button type="button" data-testid="save-lottery" :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存抽奖配置' }}</button>
      </div>
    </div>
  </details>
</template>

<style scoped>
.lottery-card { margin-bottom: 16px; padding: 20px; }
.lottery-header { align-items: center; cursor: pointer; display: flex; gap: 20px; justify-content: space-between; list-style: none; }
.lottery-header::-webkit-details-marker { display: none; }
.lottery-header > span { display: grid; gap: 5px; }
.lottery-header strong { font-size: 19px; }
.lottery-header small { color: var(--muted); line-height: 1.5; }
.lottery-summary-action { align-items: center; color: #173f73; display: inline-flex !important; flex: 0 0 auto; font-size: 14px; font-weight: 700; grid-auto-flow: column; }
.lottery-chevron { display: inline-block; font-size: 18px; transition: transform .16s ease; }
.lottery-card[open] .lottery-chevron { transform: rotate(180deg); }
.lottery-content { border-top: 1px solid var(--border); display: grid; gap: 14px; margin-top: 16px; padding-top: 16px; }
.lottery-switch { align-items: center; cursor: pointer; display: flex; flex: 0 0 auto; font-weight: 700; gap: 9px; min-height: 44px; }
.lottery-switch input { height: 1px; opacity: 0; overflow: hidden; position: absolute; width: 1px; }
.lottery-toggle { background: #94a3b8; border-radius: 999px; display: inline-flex; height: 24px; padding: 3px; transition: background-color .16s ease; width: 42px; }
.lottery-toggle::after { background: #fff; border-radius: 50%; box-shadow: 0 1px 2px rgb(15 23 42 / 18%); content: ''; height: 18px; transition: transform .16s ease; width: 18px; }
.lottery-switch input:checked + .lottery-toggle { background: #173f73; }
.lottery-switch input:checked + .lottery-toggle::after { transform: translateX(18px); }
.lottery-switch input:focus-visible + .lottery-toggle { outline: 3px solid #bfdbfe; outline-offset: 2px; }
.lottery-help { background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 7px; color: #475569; font-size: 14px; line-height: 1.6; margin: 0; padding: 10px 12px; }
.lottery-prize-list { display: grid; gap: 10px; }
.lottery-prize-row { border: 1px solid var(--border); border-radius: 8px; padding: 12px; }
.lottery-prize-head { align-items: center; display: flex; gap: 10px; justify-content: flex-start; }
.prize-enabled { color: #0f766e; cursor: pointer; font-size: 14px; }
.prize-enabled input { margin-right: 5px; }
.lottery-prize-head .danger { margin-left: auto; }
.lottery-prize-fields { display: grid; gap: 10px; grid-template-columns: minmax(150px, 2fr) minmax(90px, 1fr) minmax(125px, 1fr); margin-top: 10px; }
.lottery-prize-fields label { color: var(--muted); display: grid; font-size: 13px; gap: 5px; }
.lottery-prize-fields input { min-width: 0; }
.lottery-prize-fields label:not(.lottery-image):has(input[type='number']) { grid-template-columns: 1fr auto; align-items: end; }
.lottery-image { grid-column: 1 / -1; }
@media (max-width: 620px) {
  .lottery-card { padding: 16px; }
  .lottery-header { align-items: center; }
  .lottery-switch { justify-content: flex-start; }
  .lottery-prize-fields { grid-template-columns: 1fr 1fr; }
  .lottery-prize-fields label:first-child, .lottery-image { grid-column: 1 / -1; }
}
</style>
