// @vitest-environment jsdom
import { beforeEach, describe, it, expect, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
vi.mock('vue-router', () => ({ useRouter: () => ({ currentRoute: { value: { params: {} } } }) }))
vi.mock('@main/websocket', () => ({
  subscribeToConversation: vi.fn(),
  sendTypingIndicator: vi.fn(),
  subscribeListReplace: vi.fn()
}))
vi.mock('@main/i18n', () => ({ getI18n: () => ({ global: { t: (key) => key } }) }))
vi.mock('@main/composables/useEmitter', () => ({
  useEmitter: () => ({ emit: vi.fn(), on: vi.fn(), off: vi.fn() })
}))
const { mailboxRequest, allRequest } = vi.hoisted(() => ({
  mailboxRequest: vi.fn(),
  allRequest: vi.fn()
}))
vi.mock('@main/api', () => ({
  default: { getMailboxConversations: mailboxRequest, getAllConversations: allRequest }
}))
import { useConversationStore } from './conversation'
const response = (uuid, inboxID) => ({
  data: {
    data: {
      results: [{ uuid, inbox_id: inboxID, status: 'Open' }],
      page: 1,
      total: 1,
      total_pages: 1
    }
  }
})

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})
describe('separate mailbox lists', () => {
  it('keeps arrivals in their own inbox and preserves its scope when sorting or refreshing', async () => {
    mailboxRequest.mockResolvedValue(response('billing', 6))
    const store = useConversationStore()
    await store.fetchConversationsList(false, 'mailbox', 0, [], 0, 0, 6)
    expect(mailboxRequest).toHaveBeenCalledWith(6, expect.any(Object))
    store.handleConvPush({ uuid: 'support', inbox_id: 7, status: 'Open' })
    expect(store.conversationsList.map((row) => row.uuid)).toEqual(['billing'])
    store.handleConvPush({ uuid: 'billing-new', inbox_id: 6, status: 'Open' })
    expect(store.conversationsList.map((row) => row.uuid)).toEqual(['billing-new', 'billing'])
    await store.fetchFirstPageConversations()
    expect(mailboxRequest.mock.lastCall[0]).toBe(6)
  })
  it('clears the previous mailbox rows and ignores its delayed response after switching inboxes', async () => {
    let finishBilling
    mailboxRequest.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishBilling = resolve
        })
    )
    mailboxRequest.mockResolvedValueOnce(response('support', 7))
    const store = useConversationStore()
    const billing = store.fetchConversationsList(false, 'mailbox', 0, [], 0, 0, 6)
    await store.fetchConversationsList(false, 'mailbox', 0, [], 0, 0, 7)
    finishBilling(response('billing', 6))
    await billing
    expect(store.conversationsList.map((row) => row.uuid)).toEqual(['support'])
    allRequest.mockResolvedValue(response('all', 1))
    await store.fetchConversationsList(false, 'all')
    expect(store.conversations.inboxID).toBe(0)
    expect(allRequest).toHaveBeenCalled()
  })
})
