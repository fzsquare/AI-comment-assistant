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
      <span class="lottery-summary-action">
        <span class="lottery-action-collapsed">展开配置</span>
        <span class="lottery-action-expanded">收起配置</span>
        <span class="lottery-chevron" aria-hidden="true">⌄</span>
      </span>
    </summary>
    <div class="lottery-content" aria-labelledby="lottery-settings-title">
      <div class="lottery-status-row">
        <span class="lottery-status-copy">
          <strong>活动状态</strong>
          <small>{{ form.enabled ? '顾客完成店铺跳转并返回后，将自动参与抽奖' : '开启后，已配置的奖品才会展示给顾客' }}</small>
        </span>
        <label class="lottery-switch">
          <span>{{ form.enabled ? '已开启' : '未开启' }}</span>
          <input v-model="form.enabled" data-testid="lottery-enabled" type="checkbox" />
          <span class="lottery-toggle" aria-hidden="true"></span>
        </label>
      </div>
      <p class="lottery-help"><strong>发放方式</strong><span>中奖后由店员当场赠送，无需支付、会员或后续核销。</span></p>

      <div class="lottery-prize-list">
        <article v-for="(prize, index) in form.prizes" :key="prize.id || `new-${index}`" class="lottery-prize-row">
          <div class="lottery-prize-head">
            <span class="lottery-prize-title">
              <span class="lottery-prize-index" aria-hidden="true">{{ index + 1 }}</span>
              <span>
                <strong>奖品 {{ index + 1 }}</strong>
                <small>{{ prize.enabled ? '参与本次抽奖' : '暂不参与抽奖' }}</small>
              </span>
            </span>
            <span class="lottery-prize-actions">
              <label class="prize-enabled">
                <input v-model="prize.enabled" type="checkbox" />
                <span class="prize-enabled-control" aria-hidden="true"></span>
                <span>{{ prize.enabled ? '已启用' : '未启用' }}</span>
              </label>
              <button type="button" class="prize-remove" :aria-label="`删除奖品 ${index + 1}`" @click="removePrize(index)">删除</button>
            </span>
          </div>
          <div class="lottery-prize-fields">
            <label class="lottery-name-field"><span>奖品名称</span><input v-model.trim="prize.name" data-testid="lottery-prize-name" maxlength="64" placeholder="如：招牌小吃一份" /></label>
            <label><span>库存数量</span><span class="lottery-field-with-unit"><input v-model.number="prize.stock" type="number" inputmode="numeric" min="0" max="100000" /><span>份</span></span></label>
            <label><span>中奖概率</span><span class="lottery-field-with-unit"><input v-model.number="prize.winRate" type="number" inputmode="decimal" min="0" max="100" /><span>%</span></span></label>
            <label class="lottery-image"><span>奖品图片地址 <small>选填</small></span><input v-model.trim="prize.imageUrl" inputmode="url" placeholder="https://…" /></label>
          </div>
        </article>
      </div>
      <p v-if="form.prizes.length === 0" class="lottery-empty">还没有奖品，请先添加一个可当场赠送的奖品。</p>
      <div class="lottery-footer">
        <span class="lottery-add-action">
          <button type="button" class="secondary" data-testid="add-lottery-prize" :disabled="form.prizes.length >= 12 || saving" @click="addPrize">＋ 新增奖品</button>
          <small>{{ form.prizes.length }}/12</small>
        </span>
        <button type="button" data-testid="save-lottery" :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存抽奖配置' }}</button>
      </div>
    </div>
  </details>
</template>

<style scoped>
.lottery-card { margin-bottom: 16px; padding: 0; }
.lottery-header { align-items: center; cursor: pointer; display: flex; gap: 20px; justify-content: space-between; list-style: none; padding: 20px; }
.lottery-header::-webkit-details-marker { display: none; }
.lottery-header > span { display: grid; gap: 5px; }
.lottery-header strong { font-size: 19px; }
.lottery-header small { color: var(--muted); line-height: 1.5; }
.lottery-summary-action { align-items: center; color: var(--primary-strong); display: inline-flex !important; flex: 0 0 auto; font-size: 14px; font-weight: 700; gap: 5px; grid-auto-flow: column; }
.lottery-action-expanded { display: none; }
.lottery-card[open] .lottery-action-collapsed { display: none; }
.lottery-card[open] .lottery-action-expanded { display: inline; }
.lottery-chevron { display: inline-block; font-size: 18px; transition: transform .16s ease; }
.lottery-card[open] .lottery-chevron { transform: rotate(180deg); }
.lottery-content { border-top: 1px solid var(--border); display: grid; gap: 16px; padding: 20px; }
.lottery-status-row { align-items: center; display: flex; gap: 20px; justify-content: space-between; }
.lottery-status-copy { display: grid; gap: 3px; }
.lottery-status-copy strong { color: var(--text); font-size: 15px; }
.lottery-status-copy small { color: var(--muted); font-size: 14px; line-height: 1.5; }
.lottery-switch { align-items: center; color: var(--text-secondary); cursor: pointer; display: flex; flex: 0 0 auto; font-size: 14px; font-weight: 700; gap: 9px; min-height: 44px; }
.lottery-switch input { height: 1px; opacity: 0; overflow: hidden; position: absolute; width: 1px; }
.lottery-toggle { background: #94a3b8; border-radius: 999px; display: inline-flex; height: 24px; padding: 3px; transition: background-color .16s ease; width: 42px; }
.lottery-toggle::after { background: #fff; border-radius: 50%; box-shadow: 0 1px 2px rgb(15 23 42 / 18%); content: ''; height: 18px; transition: transform .16s ease; width: 18px; }
.lottery-switch input:checked + .lottery-toggle { background: var(--primary); }
.lottery-switch input:checked + .lottery-toggle::after { transform: translateX(18px); }
.lottery-switch input:focus-visible + .lottery-toggle { outline: 3px solid #bfdbfe; outline-offset: 2px; }
.lottery-help { align-items: baseline; background: var(--surface-subtle); border-radius: 6px; color: var(--muted); display: flex; font-size: 14px; gap: 10px; line-height: 1.6; margin: 0; padding: 10px 12px; }
.lottery-help strong { color: var(--text-secondary); flex: 0 0 auto; }
.lottery-prize-list { display: grid; gap: 12px; }
.lottery-prize-row { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
.lottery-prize-head { align-items: center; background: var(--surface-subtle); border-bottom: 1px solid var(--border-soft); display: flex; gap: 16px; justify-content: space-between; padding: 12px 16px; }
.lottery-prize-title { align-items: center; display: flex; gap: 10px; min-width: 0; }
.lottery-prize-title > span:last-child { display: grid; gap: 1px; }
.lottery-prize-title strong { color: var(--text); font-size: 15px; }
.lottery-prize-title small { color: var(--muted); font-size: 12px; }
.lottery-prize-index { align-items: center; background: #e2e8f0; border-radius: 6px; color: var(--text-secondary); display: inline-flex; flex: 0 0 auto; font-size: 13px; font-weight: 800; height: 30px; justify-content: center; width: 30px; }
.lottery-prize-actions { align-items: center; display: flex; flex: 0 0 auto; gap: 8px; }
.prize-enabled { align-items: center; color: var(--text-secondary); cursor: pointer; display: inline-flex; font-size: 13px; font-weight: 700; gap: 7px; min-height: 44px; }
.prize-enabled input { height: 1px; opacity: 0; overflow: hidden; position: absolute; width: 1px; }
.prize-enabled-control { background: #fff; border: 1px solid #94a3b8; border-radius: 4px; height: 18px; position: relative; width: 18px; }
.prize-enabled input:checked + .prize-enabled-control { background: var(--primary); border-color: var(--primary); }
.prize-enabled input:checked + .prize-enabled-control::after { border: solid #fff; border-width: 0 2px 2px 0; content: ''; height: 8px; left: 6px; position: absolute; top: 3px; transform: rotate(45deg); width: 4px; }
.prize-enabled input:focus-visible + .prize-enabled-control { outline: 3px solid #bfdbfe; outline-offset: 2px; }
.prize-remove { background: transparent; border: 1px solid transparent; color: var(--danger); font-size: 13px; font-weight: 700; padding: 8px 10px; }
.prize-remove:hover { background: var(--danger-soft); border-color: #fecaca; }
.lottery-prize-fields { display: grid; gap: 14px 12px; grid-template-columns: minmax(240px, 1fr) minmax(120px, 160px) minmax(120px, 160px); padding: 16px; }
.lottery-prize-fields label { color: var(--text-secondary); display: grid; font-size: 13px; font-weight: 700; gap: 6px; }
.lottery-prize-fields input { min-width: 0; }
.lottery-field-with-unit { position: relative; }
.lottery-field-with-unit input { padding-right: 38px; }
.lottery-field-with-unit input[type='number'] { appearance: textfield; }
.lottery-field-with-unit input[type='number']::-webkit-inner-spin-button,
.lottery-field-with-unit input[type='number']::-webkit-outer-spin-button { appearance: none; margin: 0; }
.lottery-field-with-unit > span { color: var(--muted); font-size: 13px; font-weight: 400; pointer-events: none; position: absolute; right: 12px; top: 50%; transform: translateY(-50%); }
.lottery-image { grid-column: 1 / -1; }
.lottery-image small { color: var(--muted); font-size: 12px; font-weight: 400; }
.lottery-empty { border: 1px dashed var(--border); border-radius: 8px; color: var(--muted); margin: 0; padding: 20px; text-align: center; }
.lottery-footer { align-items: center; border-top: 1px solid var(--border-soft); display: flex; justify-content: space-between; padding-top: 16px; }
.lottery-add-action { align-items: center; display: inline-flex; gap: 9px; }
.lottery-add-action small { color: var(--muted); font-variant-numeric: tabular-nums; }

@media (max-width: 760px) {
  .lottery-prize-fields { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .lottery-name-field, .lottery-image { grid-column: 1 / -1; }
}

@media (max-width: 620px) {
  .lottery-header, .lottery-content { padding: 16px; }
  .lottery-header { align-items: center; }
  .lottery-header small { font-size: 13px; }
  .lottery-status-row { align-items: flex-start; flex-direction: column; gap: 8px; }
  .lottery-switch { justify-content: space-between; width: 100%; }
  .lottery-help { align-items: flex-start; flex-direction: column; gap: 2px; }
  .lottery-prize-head { align-items: flex-start; padding: 12px; }
  .lottery-prize-actions { gap: 2px; }
  .prize-enabled > span:last-child { display: none; }
  .lottery-prize-fields { grid-template-columns: 1fr; padding: 12px; }
  .lottery-name-field, .lottery-image { grid-column: auto; }
  .lottery-footer { align-items: stretch; flex-direction: column; gap: 12px; }
  .lottery-footer > button { width: 100%; }
  .lottery-add-action { justify-content: space-between; }
}
</style>
