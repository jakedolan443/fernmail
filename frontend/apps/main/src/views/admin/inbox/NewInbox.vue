<template>
  <EmailInboxForm
    :initial-values="{}"
    :submitForm="submitForm"
    :isLoading="isLoading"
    :isNewForm="true"
  />
</template>

<script setup>
import { ref } from 'vue'

import { useRouter } from 'vue-router'

import EmailInboxForm from '@/features/admin/inbox/EmailInboxForm.vue'
import api from '../../../api'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const emitter = useEmitter()
const isLoading = ref(false)

const router = useRouter()

const submitForm = (values) => {
  const channelName = 'email'
  const payload = {
    name: values.name,
    from: values.from,
    from_name_template: values.from_name_template || '',
    channel: channelName,
    enabled: values.enabled ?? true,

    config: {
      reply_to: values.reply_to,
      enable_plus_addressing: values.enable_plus_addressing,
      imap: [values.imap],
      smtp: [values.smtp]
    }
  }
  createInbox(payload)
}

async function createInbox(payload) {
  try {
    isLoading.value = true
    await api.createInbox(payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    router.push({ name: 'inbox-list' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}
</script>
