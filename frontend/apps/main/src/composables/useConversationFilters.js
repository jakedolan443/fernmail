import { computed } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useInboxStore } from '@/stores/inbox'
import { FIELD_TYPE, FIELD_OPERATORS } from '@/constants/filterConfig'
import { useI18n } from 'vue-i18n'
export function useConversationFilters() {
  const cStore = useConversationStore()
  const iStore = useInboxStore()
  const { t } = useI18n()
  const conversationsListFilters = computed(() => ({
    status_id: {
      label: t('globals.terms.status'),
      type: FIELD_TYPE.SELECT,
      operators: FIELD_OPERATORS.SELECT,
      options: cStore.statusOptions
    },

    inbox_id: {
      label: t('globals.terms.inbox'),
      type: FIELD_TYPE.SELECT,
      operators: FIELD_OPERATORS.SELECT,
      options: iStore.options
    },
    email_alias: {
      label: t('conversation.emailAlias'),
      type: FIELD_TYPE.TEXT,
      operators: FIELD_OPERATORS.TEXT_EXACT
    },
    created_at: {
      label: t('globals.terms.createdAt'),
      type: FIELD_TYPE.DATE,
      operators: FIELD_OPERATORS.DATE
    },
    waiting_since: {
      label: t('globals.terms.waitingSince'),
      type: FIELD_TYPE.DATE,
      operators: FIELD_OPERATORS.DATE
    },
    snoozed_until: {
      label: t('globals.terms.snoozedUntil'),
      type: FIELD_TYPE.DATE,
      operators: FIELD_OPERATORS.DATE
    },
    last_message_at: {
      label: t('globals.terms.lastMessageAt'),
      type: FIELD_TYPE.DATE,
      operators: FIELD_OPERATORS.DATE
    },
    last_interaction_at: {
      label: t('globals.terms.lastInteractionAt'),
      type: FIELD_TYPE.DATE,
      operators: FIELD_OPERATORS.DATE
    },

    email: {
      label: t('globals.terms.contactEmail'),
      type: FIELD_TYPE.TEXT,
      operators: FIELD_OPERATORS.TEXT,
      model: 'users'
    },

    last_interaction_sender: {
      label: t('globals.terms.lastInteractionBy'),
      type: FIELD_TYPE.SELECT,
      operators: FIELD_OPERATORS.SELECT,
      options: [
        { label: t('globals.terms.correspondent'), value: 'contact' },
        { label: t('globals.terms.agent'), value: 'agent' }
      ]
    }
  }))

  return { conversationsListFilters }
}
