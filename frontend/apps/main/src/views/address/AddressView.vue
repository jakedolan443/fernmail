<template>
  <ConversationPlaceholder v-if="!addressID" />
  <router-view v-else />
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useIntervalFn, useDocumentVisibility } from '@vueuse/core'
import { useConversationStore } from '@main/stores/conversation'
import { useAddressStore } from '@main/stores/address'
import ConversationPlaceholder from '@/features/conversation/ConversationPlaceholder.vue'

const route = useRoute()
const router = useRouter()
const conversationStore = useConversationStore()
const addressStore = useAddressStore()
const addressID = computed(() => route.params.addressID)

let lastFetchedAddressID = ''

const hasCurrentList = () =>
  conversationStore.conversations.initialized &&
  conversationStore.conversations.addressID === Number(addressID.value)

async function fetchForCurrentRoute() {
  await addressStore.fetchAddresses()
  if (!addressID.value) {
    const first = addressStore.addresses[0]
    if (first) router.replace({ name: 'address-inbox', params: { addressID: first.id } })
    return
  }

  if (!addressStore.get(addressID.value)) {
    router.replace({ name: 'address-inbox' })
    return
  }

  if (String(addressID.value) === lastFetchedAddressID && hasCurrentList()) {
    conversationStore.refreshConversationList()
    return
  }
  lastFetchedAddressID = String(addressID.value)
  conversationStore.fetchConversationsList(true, Number(addressID.value))
}

onMounted(fetchForCurrentRoute)
watch(addressID, fetchForCurrentRoute)

const visibility = useDocumentVisibility()
const { pause, resume } = useIntervalFn(() => conversationStore.refreshConversationList(), 120000)
watch(visibility, (value) => {
  if (value === 'visible') {
    conversationStore.refreshConversationList()
    resume()
  } else {
    pause()
  }
})
</script>
