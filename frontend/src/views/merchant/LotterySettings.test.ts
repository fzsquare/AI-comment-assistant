import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import LotterySettings from './LotterySettings.vue'

describe('LotterySettings', () => {
  it('lets a merchant enable the activity and save immediate-gift prizes', async () => {
    const wrapper = mount(LotterySettings, {
      props: { config: { enabled: false, prizes: [] }, saving: false }
    })

    expect(wrapper.get('details').attributes('open')).toBeUndefined()
    await wrapper.get('summary').trigger('click')
    await wrapper.get('[data-testid="lottery-enabled"]').setValue(true)
    await wrapper.get('[data-testid="add-lottery-prize"]').trigger('click')
    await wrapper.get('[data-testid="lottery-prize-name"]').setValue('招牌小吃一份')
    await wrapper.get('[data-testid="save-lottery"]').trigger('click')

    expect(wrapper.emitted('save')?.[0]?.[0]).toMatchObject({
      enabled: true,
      prizes: [{ name: '招牌小吃一份', stock: 10, winRate: 10, enabled: true }]
    })
  })
})
