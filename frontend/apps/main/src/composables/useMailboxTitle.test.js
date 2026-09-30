// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, reactive, ref } from 'vue'

const state = vi.hoisted(() => ({
  route: null,
  settings: null,
  conversations: null,
  inboxes: null,
  locale: null
}))
vi.mock('vue-router', () => ({ useRoute: () => state.route }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key, locale: state.locale }) }))
vi.mock('@main/stores/appSettings', () => ({ useAppSettingsStore: () => state.settings }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => state.conversations }))
vi.mock('@main/stores/inbox', () => ({ useInboxStore: () => state.inboxes }))
import { useMailboxTitle } from './useMailboxTitle'

let scope
beforeEach(() => {
  state.route = reactive({
    fullPath: '/inboxes/all',
    params: { type: 'all' },
    meta: { typeKey: () => 'All mail' }
  })
  state.settings = reactive({ settings: { 'app.site_name': 'Fernmail' } })
  state.conversations = reactive({ sidebarCounts: { unread: 4 } })
  state.inboxes = reactive({ inboxes: [] })
  state.locale = ref('en-US')
  scope = effectScope()
  scope.run(useMailboxTitle)
})
afterEach(() => scope.stop())

describe('mailbox tab title', () => {
  it('uses the selected mailbox name after inboxes load', async () => {
    state.route.params.inboxID = '2'
    state.route.fullPath = '/inboxes/mailbox/2'
    state.inboxes.inboxes = [{ id: 2, name: 'Billing' }]
    await nextTick()
    expect(document.title).toBe('(4) Billing - Fernmail')
  })
  it('uses the global total and removes the prefix when everything is read', async () => {
    expect(document.title).toBe('(4) All mail - Fernmail')
    state.conversations.sidebarCounts.unread = 1
    await nextTick()
    expect(document.title).toBe('(1) All mail - Fernmail')
    state.conversations.sidebarCounts.unread = 0
    await nextTick()
    expect(document.title).toBe('All mail - Fernmail')
  })
  it('restores the prefix after navigation even when route metadata and count are unchanged', async () => {
    document.title = 'All mail - Fernmail' // router navigation guard
    state.route.fullPath = '/inboxes/all/conversation/another-mail'
    await nextTick()
    expect(document.title).toBe('(4) All mail - Fernmail')
    state.route.meta = { titleKey: 'Settings' }
    state.route.fullPath = '/admin/general'
    state.settings.settings['app.site_name'] = 'Studio mail'
    await nextTick()
    expect(document.title).toBe('(4) Settings - Studio mail')
  })
})
