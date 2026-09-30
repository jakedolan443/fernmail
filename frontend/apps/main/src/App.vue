<template>
  <div class="flex w-full h-dvh text-foreground bg-canvas p-1 md:p-1.5">
    <!-- Main sidebar that collapses -->
    <div class="flex-1 min-w-0">
      <Sidebar
        :userViews="viewStore.views"
        :sharedViews="sharedViewStore.sharedViewList"
        @create-view="createView"
        @edit-view="editView"
        @delete-view="deleteView"
        @create-conversation="openCreateConversation()"
      >
        <div class="flex flex-col h-full rounded-lg overflow-hidden bg-background">
          <ConnectionBanner />

          <!-- Show admin banner only in admin routes -->
          <AdminBanner v-if="route.path.startsWith('/admin')" />

          <!-- Common header for all pages -->
          <PageHeader />

          <!-- Main content -->
          <RouterView class="flex-grow" />
        </div>
        <ViewForm v-model:openDialog="openCreateViewForm" v-model:view="view" />
      </Sidebar>
    </div>
  </div>

  <!-- Command box -->
  <Command />

  <!-- Create conversation dialog -->
  <CreateConversation v-if="openCreateConversationDialog" v-model="openCreateConversationDialog" />
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { retireNotificationWorker } from './composables/retireNotificationWorker'
import { useStorage, useEventListener } from '@vueuse/core'
import { RouterView } from 'vue-router'
import { useUserStore } from './stores/user'
import { initWS } from './websocket.js'
import { EMITTER_EVENTS } from './constants/emitterEvents.js'
import { useEmitter } from './composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useConversationStore } from './stores/conversation'
import { CONVERSATION_LIST_TYPE } from './constants/conversation'
import { useInboxStore } from './stores/inbox'
import { useUsersStore } from './stores/users'
import { useSharedViewStore } from './stores/sharedView'
import { useIdleDetection } from './composables/useIdleDetection'
import { useMailboxTitle } from './composables/useMailboxTitle'
import { useViewStore } from './stores/view'
import PageHeader from './components/layout/PageHeader.vue'
import ViewForm from '@/features/view/ViewForm.vue'
import AdminBanner from '@/components/banner/AdminBanner.vue'
import ConnectionBanner from '@/components/banner/ConnectionBanner.vue'
import { toast as sooner } from 'vue-sonner'
import Sidebar from '@main/components/sidebar/Sidebar.vue'
import Command from '@/features/command/CommandBox.vue'
import CreateConversation from '@/features/conversation/CreateConversation.vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'

import api from '@main/api'

const route = useRoute()
const emitter = useEmitter()

// Remember last inbox path so navigating back from admin/contacts/reports restores it
const lastInboxPath = useStorage('lastInboxPath', '')
const userStore = useUserStore()
const conversationStore = useConversationStore()
watch(
  () => route.path,
  (path) => {
    if (path.startsWith('/inboxes') && path !== '/inboxes/search') {
      lastInboxPath.value = path
    }
  },
  { immediate: true }
)

// Opening a conversation inside an inbox does not change the inbox path used here.
watch(
  () => route.path.replace(/\/conversation\/.*$/, ''),
  (inboxPath) => {
    if (inboxPath.startsWith('/inboxes') && inboxPath !== '/inboxes/search') {
      conversationStore.fetchSidebarCounts()
    }
  }
)
const usersStore = useUsersStore()
const inboxStore = useInboxStore()
const sharedViewStore = useSharedViewStore()
const viewStore = useViewStore()
const view = ref({})
const openCreateViewForm = ref(false)
const openCreateConversationDialog = ref(false)
const createConversationContact = ref(null)
const { t } = useI18n()
initWS()
useIdleDetection()
useMailboxTitle()
useEventListener(window, 'focus', () => conversationStore.fetchSidebarCounts({ force: true }))

onMounted(() => {
  retireNotificationWorker()
  initToaster()
  listenViewRefresh()
  emitter.on(EMITTER_EVENTS.OPEN_CREATE_CONVERSATION, openCreateConversation)
  emitter.on(EMITTER_EVENTS.OPEN_VIEW_FORM, createView)
  initStores()
})

const openCreateConversation = ({ contact = null } = {}) => {
  createConversationContact.value = contact
  openCreateConversationDialog.value = true
}

// Initialize data stores
const initStores = async () => {
  if (!userStore.userID) {
    await userStore.getCurrentUser()
  }
  await Promise.allSettled([
    viewStore.fetchViews(),
    sharedViewStore.loadSharedViews(),
    conversationStore.fetchStatuses(),
    conversationStore.fetchAllDrafts(),
    usersStore.fetchUsers(),
    inboxStore.fetchInboxes()
  ])
}

const createView = () => {
  view.value = {}
  openCreateViewForm.value = true
}

const editView = (v) => {
  view.value = { ...v }
  openCreateViewForm.value = true
}

const deleteView = async (view) => {
  try {
    await api.deleteView(view.id)
    emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'view' })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.deletedSuccessfully')
    })
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  }
}

const initToaster = () => {
  emitter.on(EMITTER_EVENTS.SHOW_TOAST, (message) => {
    if (!message.description) return
    if (message.variant === 'destructive') {
      sooner.error(message.description)
    } else if (message.variant === 'warning') {
      sooner.warning(message.description)
    } else if (message.variant === 'info') {
      sooner.info(message.description)
    } else {
      sooner.success(message.description)
    }
  })
}

const listenViewRefresh = () => {
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshViews)
}

const refreshViews = async (data) => {
  openCreateViewForm.value = false
  // TODO: move model to constants.
  if (data?.model === 'view') {
    await viewStore.fetchViews()
    if (data.id) conversationStore.fetchViewCount(data.id)
    else conversationStore.fetchSidebarCounts({ force: true })
    const openID = route.params.viewID
    // If the open view was edited its filters may have changed, refetch.
    if (openID && viewStore.views.some((v) => String(v.id) === String(openID))) {
      // Reset list and fetch conversations.
      conversationStore.resetConversations()
      conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], openID)
    }
  }
}
</script>

<style scoped>
:deep(.group\/sidebar-wrapper) {
  min-height: auto !important;
  height: 100%;
}
</style>
