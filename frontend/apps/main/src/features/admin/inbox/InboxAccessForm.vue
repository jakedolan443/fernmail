<script setup>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ShieldCheck } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

const props = defineProps({ inboxID: { type: Number, required: true } })
const { t } = useI18n()
const emitter = useEmitter()
const access = ref(null)
const loading = ref(true)
const saving = ref(false)
const error = ref('')

watch(
  () => props.inboxID,
  async (id) => {
    loading.value = true
    error.value = ''
    try {
      const response = await api.getInboxAccess(id)
      access.value = response.data.data
    } catch (err) {
      error.value = handleHTTPError(err).message
    } finally {
      loading.value = false
    }
  },
  { immediate: true }
)

async function save() {
  saving.value = true
  error.value = ''
  try {
    await api.updateInboxAccess(props.inboxID, {
      restricted: access.value.restricted,
      user_ids: access.value.user_ids,
      role_ids: access.value.role_ids
    })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('inbox.access.saved') })
  } catch (err) {
    error.value = handleHTTPError(err).message
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="mb-8 rounded-lg border p-5" aria-labelledby="inbox-access-title">
    <div class="mb-1 flex items-center gap-2">
      <ShieldCheck class="size-4 text-emerald-600" />
      <h2 id="inbox-access-title" class="font-semibold">{{ t('inbox.access.title') }}</h2>
    </div>
    <p class="mb-4 text-sm text-muted-foreground">{{ t('inbox.access.description') }}</p>
    <Spinner v-if="loading" />
    <form v-else-if="access" class="space-y-4" @submit.prevent="save">
      <fieldset class="space-y-2" :disabled="saving">
        <legend class="sr-only">{{ t('inbox.access.title') }}</legend>
        <label class="flex items-center gap-2 text-sm cursor-pointer">
          <input
            v-model="access.restricted"
            type="radio"
            :value="false"
            class="accent-emerald-600"
          />
          {{ t('inbox.access.everyone') }}
        </label>
        <label class="flex items-center gap-2 text-sm cursor-pointer">
          <input
            v-model="access.restricted"
            type="radio"
            :value="true"
            class="accent-emerald-600"
          />
          {{ t('inbox.access.selected') }}
        </label>
      </fieldset>
      <div v-if="access.restricted" class="grid gap-4 sm:grid-cols-2">
        <fieldset class="rounded-md border p-3" :disabled="saving">
          <legend class="px-1 text-sm font-medium">{{ t('inbox.access.users') }}</legend>
          <div class="max-h-48 space-y-2 overflow-y-auto">
            <label
              v-for="user in access.users"
              :key="user.id"
              class="flex items-start gap-2 text-sm cursor-pointer"
            >
              <input
                v-model="access.user_ids"
                type="checkbox"
                :value="user.id"
                class="mt-1 accent-emerald-600"
              />
              <span class="break-all">{{ user.name }}</span>
            </label>
            <p v-if="!access.users?.length" class="text-sm text-muted-foreground">
              {{ t('inbox.access.noUsers') }}
            </p>
          </div>
        </fieldset>
        <fieldset class="rounded-md border p-3" :disabled="saving">
          <legend class="px-1 text-sm font-medium">{{ t('inbox.access.roles') }}</legend>
          <div class="max-h-48 space-y-2 overflow-y-auto">
            <label
              v-for="role in access.roles"
              :key="role.id"
              class="flex items-center gap-2 text-sm cursor-pointer"
            >
              <input
                v-model="access.role_ids"
                type="checkbox"
                :value="role.id"
                class="accent-emerald-600"
              />
              {{ role.name }}
            </label>
          </div>
        </fieldset>
      </div>
      <p v-if="access.restricted" class="text-sm text-muted-foreground">
        {{ t('inbox.access.union') }}
      </p>
      <p class="text-xs text-muted-foreground">{{ t('inbox.access.admins') }}</p>
      <Button type="submit" :disabled="saving">{{
        t(saving ? 'inbox.access.saving' : 'inbox.access.save')
      }}</Button>
    </form>
    <p v-if="error" role="alert" class="mt-3 text-sm text-destructive">{{ error }}</p>
  </section>
</template>
