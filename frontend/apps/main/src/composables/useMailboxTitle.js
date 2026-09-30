import { watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppSettingsStore } from '@main/stores/appSettings'
import { useInboxStore } from '@main/stores/inbox'
import { useConversationStore } from '@main/stores/conversation'

export function useMailboxTitle() {
  const route = useRoute()
  const { t, locale } = useI18n()
  const settings = useAppSettingsStore()
  const conversations = useConversationStore()
  const inboxes = useInboxStore()

  const updateTitle = () => {
    const typeKey = typeof route.meta?.typeKey === 'function' ? route.meta.typeKey(route) : ''
    const titleKey = typeKey || route.meta?.titleKey
    const mailbox = inboxes.inboxes.find(
      (item) => String(item.id) === String(route.params?.inboxID)
    )
    const page =
      mailbox?.name ||
      (titleKey ? t(titleKey, route.meta?.titleCount || 1) : t('globals.terms.inbox'))
    const site = settings.settings['app.site_name'] || 'Fernmail'
    const count = conversations.sidebarCounts.unread
    document.title = `${count > 0 ? `(${count}) ` : ''}${page} - ${site}`
  }
  watch(
    [
      () => route.fullPath,
      () => inboxes.inboxes,
      () => route.meta,
      () => settings.settings['app.site_name'],
      () => conversations.sidebarCounts.unread,
      locale
    ],
    updateTitle,
    { immediate: true }
  )
}
