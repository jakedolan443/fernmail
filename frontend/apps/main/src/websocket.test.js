// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { conversations, notifyNewMessage } = vi.hoisted(() => ({
  conversations: {
    conversation: { data: null },
    sidebarCounts: { unread: 0 },
    handleConvPush: vi.fn(),
    mergeConversationUpdate: vi.fn(),
    incrementUnread: vi.fn(),
    isConversationInList: vi.fn(() => true),
    updateConversationMessage: vi.fn(),
    refreshSidebarCounts: vi.fn()
  },
  notifyNewMessage: vi.fn()
}))
vi.mock('./stores/conversation', () => ({ useConversationStore: () => conversations }))
vi.mock('./stores/browserNotifications', () => ({
  useBrowserNotificationsStore: () => ({ notifyNewMessage })
}))
vi.mock('./stores/users', () => ({ useUsersStore: () => ({}) }))
vi.mock('./stores/connection', () => ({ useConnectionStore: () => ({}) }))
vi.mock('./composables/useEmitter', () => ({ useEmitter: () => ({}) }))
import { WebSocketClient } from './websocket'

const mail = {
  uuid: 'mail-1',
  conversation_uuid: 'thread-1',
  type: 'incoming',
  sender_type: 'contact',
  preview: 'Hello'
}
let client
beforeEach(() => {
  vi.clearAllMocks()
  conversations.conversation.data = null
  conversations.sidebarCounts.unread = 0
  Object.defineProperty(document, 'hidden', { configurable: true, value: false })
  client = new WebSocketClient()
  client.socket = {}
})
const receive = (type, data) =>
  client.handleMessage({ target: client.socket, data: JSON.stringify({ type, data }) })

describe('live mail delivery', () => {
  it('updates the inbox and sends each new-message event to browser notifications', () => {
    receive('new_message', mail)
    expect(conversations.incrementUnread).toHaveBeenCalledWith('thread-1')
    expect(conversations.updateConversationMessage).toHaveBeenCalledWith(mail)
    expect(notifyNewMessage).toHaveBeenCalledWith(mail)
  })
  it('keeps a visible open thread read, but marks arrivals unread in a background tab', () => {
    conversations.conversation.data = { uuid: 'thread-1' }
    receive('new_message', mail)
    expect(conversations.incrementUnread).not.toHaveBeenCalled()
    Object.defineProperty(document, 'hidden', { configurable: true, value: true })
    receive('new_message', mail)
    expect(conversations.incrementUnread).toHaveBeenCalledWith('thread-1')
  })
  it('does not duplicate alerts for new-conversation broadcasts or stale sockets', () => {
    receive('new_conversation', { uuid: 'thread-1' })
    client.handleMessage({ target: {}, data: JSON.stringify({ type: 'new_message', data: mail }) })
    expect(notifyNewMessage).not.toHaveBeenCalled()
  })
})
