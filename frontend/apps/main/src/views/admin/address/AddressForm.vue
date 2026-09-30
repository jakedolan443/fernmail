<template>
  <div class="mx-auto max-w-3xl">
    <div class="mb-5 flex items-center justify-between gap-4">
      <router-link :to="{ name: 'address-list' }" class="text-sm text-muted-foreground hover:text-foreground">← {{ $t('address.title', 2) }}</router-link>
    </div>
    <LoadingOverlay :loading="loading" reserve-height>
      <form class="space-y-6" @submit.prevent="save">
        <section class="space-y-4 rounded-lg border p-5">
          <div>
            <h2 class="font-semibold">{{ form.id ? $t('address.edit') : $t('address.new') }}</h2>
            <p class="mt-1 text-sm text-muted-foreground">{{ $t('address.transportHelp') }}</p>
          </div>
          <label class="block space-y-1.5">
            <span class="text-sm font-medium">{{ $t('address.address') }}</span>
            <input v-model.trim="form.address" type="email" required class="w-full rounded-md border bg-background px-3 py-2 disabled:cursor-not-allowed disabled:opacity-70" :disabled="isMailbox" placeholder="support@example.com" />
            <span v-if="isMailbox" class="text-xs text-muted-foreground">{{ $t('address.transportManaged') }}</span>
          </label>
          <label class="block space-y-1.5">
            <span class="text-sm font-medium">{{ $t('address.displayName') }} <span class="font-normal text-muted-foreground">(optional)</span></span>
            <input v-model.trim="form.display_name" type="text" maxlength="140" class="w-full rounded-md border bg-background px-3 py-2" placeholder="Support" />
          </label>
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-1.5">
              <span class="text-sm font-medium">{{ $t('address.kind') }}</span>
              <p class="rounded-md border bg-muted/30 px-3 py-2 text-sm">{{ isMailbox ? $t('address.mailbox') : $t('address.alias') }}</p>
            </div>
            <label class="block space-y-1.5">
              <span class="text-sm font-medium">{{ $t('address.selectTransport') }}</span>
              <select v-model.number="form.inbox_id" required class="w-full rounded-md border bg-background px-3 py-2 disabled:cursor-not-allowed disabled:opacity-70" :disabled="isMailbox">
                <option :value="0" disabled>Select a transport</option>
                <option v-for="inbox in emailInboxes" :key="inbox.id" :value="inbox.id">{{ inbox.from || inbox.name }}</option>
              </select>
            </label>
          </div>
          <label class="flex cursor-pointer items-center gap-2 text-sm"><input v-model="form.enabled" type="checkbox" class="accent-primary" />{{ $t('address.enabled') }}</label>
        </section>

        <section class="space-y-4 rounded-lg border p-5">
          <div>
            <h2 class="font-semibold">{{ $t('address.access') }}</h2>
            <p class="mt-1 text-sm text-muted-foreground">{{ $t('address.accessDescription') }}</p>
          </div>
          <label class="flex cursor-pointer items-center gap-2 text-sm"><input v-model="form.restricted" type="checkbox" class="accent-primary" />{{ $t('address.restricted') }}</label>
          <div v-if="form.restricted" class="grid gap-4 sm:grid-cols-2">
            <fieldset class="rounded-md border p-3">
              <legend class="px-1 text-sm font-medium">Agents</legend>
              <label v-for="user in principals.users" :key="user.id" class="flex cursor-pointer items-start gap-2 py-1 text-sm"><input v-model="form.user_ids" type="checkbox" :value="user.id" class="mt-1 accent-primary" /><span>{{ user.name }}</span></label>
            </fieldset>
            <fieldset class="rounded-md border p-3">
              <legend class="px-1 text-sm font-medium">Teams</legend>
              <label v-for="team in principals.teams" :key="team.id" class="flex cursor-pointer items-center gap-2 py-1 text-sm"><input v-model="form.team_ids" type="checkbox" :value="team.id" class="accent-primary" /><span>{{ team.name }}</span></label>
            </fieldset>
          </div>
        </section>

        <div class="flex justify-end gap-2"><Button variant="outline" type="button" @click="router.push({ name: 'address-list' })">{{ $t('globals.messages.cancel') }}</Button><Button type="submit" :disabled="saving">{{ saving ? 'Saving…' : $t('globals.messages.save') }}</Button></div>
      </form>
    </LoadingOverlay>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Button } from '@shared-ui/components/ui/button'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

const props = defineProps({ id: { type: String, default: '' } })
const router = useRouter()
const emitter = useEmitter()
const loading = ref(true)
const saving = ref(false)
const emailInboxes = ref([])
const principals = ref({ users: [], teams: [] })
const form = ref({ id: 0, address: '', display_name: '', inbox_id: 0, kind: 'alias', enabled: true, restricted: true, user_ids: [], team_ids: [] })

const isEditing = computed(() => Boolean(props.id))
const isMailbox = computed(() => form.value.kind === 'mailbox')

async function load() {
  loading.value = true
  try {
    const requests = [api.getInboxes(), api.getAddressPrincipals()]
    if (isEditing.value) requests.push(api.getAdminAddress(props.id), api.getAddressAccess(props.id))
    const responses = await Promise.all(requests)
    emailInboxes.value = (responses[0].data?.data || []).filter((inbox) => inbox.channel === 'email')
    principals.value = responses[1].data?.data || principals.value
    if (isEditing.value) {
      const address = responses[2].data?.data
      const access = responses[3].data?.data
      form.value = { ...form.value, ...address, user_ids: access.user_ids || [], team_ids: access.team_ids || [] }
      principals.value = access
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const payload = {
      ...form.value,
      kind: isEditing.value ? form.value.kind : 'alias',
      user_ids: form.value.user_ids.map(Number),
      team_ids: form.value.team_ids.map(Number)
    }
    if (isEditing.value) await api.updateAddress(props.id, payload)
    else await api.createAddress(payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Address saved' })
    router.push({ name: 'address-list' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
