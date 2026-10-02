<template>
  <AlertDialog :open="showContactEmailWarning" @update:open="showContactEmailWarning = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('replyBox.contactEmailMissing') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{
            $t('replyBox.contactEmailMissingDescription', {
              email: conversationStore.current?.correspondent?.email
            })
          }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="processSend(true, deferredStatus)">{{
          $t('replyBox.sendAnyway')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <div class="flex h-full min-h-0 flex-col overflow-hidden text-foreground bg-background">
    <div v-if="newerReply && !isEditorFullscreen" class="flex shrink-0 items-center gap-2 border-b border-warning/40 bg-warning/10 px-3 py-2 text-sm" role="status">
      <span class="flex-1">{{ t('replyBox.newerReply') }}</span>
      <Button type="button" size="sm" variant="ghost" @click="acknowledgeReply">{{ t('replyBox.reviewedReply') }}</Button>
    </div>
    <!-- Fullscreen editor -->
    <Dialog :open="isEditorFullscreen" @update:open="isEditorFullscreen = false">
      <DialogContent
        class="bg-card text-card-foreground p-4 flex flex-col overflow-hidden"
        :class="[
          isCramped
            ? 'top-0 left-0 translate-x-0 translate-y-0 w-full max-w-none h-[var(--visual-viewport-height,100dvh)] max-h-none rounded-none'
            : 'max-w-[60%] h-[70%] max-h-[75%] rounded-lg',
          { '!bg-private': messageType === 'private_note' }
        ]"
        @escapeKeyDown="isEditorFullscreen = false"
        :hide-close-button="true"
      >
        <ReplyBoxContent
          v-if="isEditorFullscreen"
          ref="fullscreenContentRef"
          :isFullscreen="true"
          :newerReply="newerReply"
          @reviewedReply="acknowledgeReply"
          :isSending="isSending"
          :isDraftLoading="isDraftLoading"
          :draftSaveState="draftSaveState"
          @retryDraftSave="retryDraftSave"
          :uploadingFiles="uploadingFiles"
          :uploadedFiles="mediaFiles"
          v-model:htmlContent="htmlContent"
          v-model:textContent="textContent"
          v-model:to="to"
          v-model:cc="cc"
          v-model:bcc="bcc"
          v-model:emailErrors="emailErrors"
          v-model:messageType="messageType"
          v-model:showBcc="showBcc"
          v-model:mentions="mentions"
          @toggleFullscreen="isEditorFullscreen = !isEditorFullscreen"
          @send="processSend"
          @sendAndSetStatus="processSendAndSetStatus"
          @fileUpload="handleFileUpload"
          @fileDelete="handleFileDelete"
          @filesDropped="uploadFiles"
          :canSendReply="canSendReply"
          :canSendPrivateNote="canSendPrivateNote"
          class="h-full flex-grow"
        />
      </DialogContent>
    </Dialog>

    <div v-if="isCramped && !isEditorFullscreen" class="p-2">
      <div v-if="draftSaveState === 'error'" class="mb-2 text-xs text-destructive" role="status">
        {{ t('replyBox.draftSaveFailed') }}
        <Button type="button" size="sm" variant="link" @click="retryDraftSave">{{ t('replyBox.retryDraftSave') }}</Button>
      </div>
      <Button
        type="button"
        variant="outline"
        class="w-full h-11 justify-start font-normal min-w-0"
        :class="{ '!bg-private': messageType === 'private_note' }"
        @click="isEditorFullscreen = true"
      >
        <Pencil class="shrink-0 text-muted-foreground" />
        <span v-if="draftPreview" class="truncate">{{ draftPreview }}</span>
        <span v-else class="truncate text-muted-foreground">
          {{
            messageType === 'private_note'
              ? $t('globals.terms.privateNote')
              : $t('globals.terms.reply')
          }}
        </span>
        <span
          v-if="attachmentCount"
          class="ml-auto flex shrink-0 items-center gap-1 text-xs text-muted-foreground"
        >
          <Paperclip class="w-3.5 h-3.5" />
          {{ attachmentCount }}
        </span>
      </Button>
    </div>

    <!-- Main Editor non-fullscreen -->
    <div
      class="bg-background text-card-foreground box m-2 flex-1 min-h-0 px-2 pt-2 flex flex-col relative overflow-hidden"
      :class="{ '!bg-private': messageType === 'private_note' }"
      v-if="!isCramped && !isEditorFullscreen"
    >
      <ReplyBoxContent
        ref="replyBoxContentRef"
        :isFullscreen="false"
        :isSending="isSending"
        :isDraftLoading="isDraftLoading"
        :draftSaveState="draftSaveState"
        @retryDraftSave="retryDraftSave"
        :uploadingFiles="uploadingFiles"
        :uploadedFiles="mediaFiles"
        v-model:htmlContent="htmlContent"
        v-model:textContent="textContent"
        v-model:to="to"
        v-model:cc="cc"
        v-model:bcc="bcc"
        v-model:emailErrors="emailErrors"
        v-model:messageType="messageType"
        v-model:showBcc="showBcc"
        v-model:mentions="mentions"
        @toggleFullscreen="isEditorFullscreen = !isEditorFullscreen"
        @send="processSend"
        @sendAndSetStatus="processSendAndSetStatus"
        @fileUpload="handleFileUpload"
        @fileDelete="handleFileDelete"
        @filesDropped="uploadFiles"
        :canSendReply="canSendReply"
        :canSendPrivateNote="canSendPrivateNote"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed, nextTick, onMounted, onUnmounted } from 'vue'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useUserStore } from '@main/stores/user'
import { useDraftManager } from '@main/composables/useDraftManager'
import { useReplyAwareness } from '@main/composables/useReplyAwareness'
import api from '@main/api'
import { useI18n } from 'vue-i18n'
import { useConversationStore } from '@main/stores/conversation'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import { Dialog, DialogContent } from '@shared-ui/components/ui/dialog'
import { Button } from '@shared-ui/components/ui/button'

import { Pencil, Paperclip } from 'lucide-vue-next'
import { useVisualViewportHeight } from '@main/composables/useVisualViewportHeight'
import { useIsComposerCramped } from '@main/composables/useIsComposerCramped'
import { useEmitter } from '@main/composables/useEmitter'
import { useFileUpload } from '@main/composables/useFileUpload'
import { hasInlineImage, hasPendingInlineUpload } from '@main/composables/useInlineImageUpload'
import ReplyBoxContent from '@/features/conversation/ReplyBoxContent.vue'
import { UserTypeAgent } from '@/constants/user'
import { permissions as perms } from '@main/constants/permissions.js'

const { t } = useI18n()
const conversationStore = useConversationStore()

const emitter = useEmitter()
const userStore = useUserStore()
const isCramped = useIsComposerCramped()
useVisualViewportHeight()

const canSendReply = computed(() => userStore.can(perms.MESSAGES_WRITE))
const canSendPrivateNote = computed(() => userStore.can(perms.MESSAGES_WRITE_PRIVATE))
const defaultMessageType = computed(() => (canSendReply.value ? 'reply' : 'private_note'))
const isAllowedMessageType = (type) =>
  (type === 'reply' && canSendReply.value) || (type === 'private_note' && canSendPrivateNote.value)
const resolveAllowedDraftType = (uuid) => {
  const type = conversationStore.resolveDraftType(uuid)
  return isAllowedMessageType(type) ? type : defaultMessageType.value
}

// Setup file upload composable
const {
  uploadingFiles,
  handleFileUpload,
  handleFileDelete,
  uploadFiles,
  mediaFiles,
  setMediaFiles
} = useFileUpload({
  linkedModel: 'messages'
})

const messageType = ref('reply')
const currentConversationUUID = computed(() => conversationStore.current?.uuid || null)
watch(
  currentConversationUUID,
  async (uuid, prevUuid) => {
    if (prevUuid) conversationStore.setSelectedDraftType(prevUuid, messageType.value)
    if (!uuid) {
      messageType.value = defaultMessageType.value
      return
    }
    const initialType = resolveAllowedDraftType(uuid)
    messageType.value = initialType
    // Prefetch may still be in flight on first load; re-resolve once drafts land.
    await conversationStore.draftsReady
    if (uuid !== currentConversationUUID.value || messageType.value !== initialType) return
    messageType.value = resolveAllowedDraftType(uuid)
  },
  { immediate: true }
)

const recipientDefaults = computed(() => ({
  uuid: currentConversationUUID.value,
  ready: conversationStore.messages.data.hasConversation(currentConversationUUID.value),
  recipients: {
    to: conversationStore.currentTo.join(', '),
    cc: conversationStore.currentCC.join(', '),
    bcc: conversationStore.currentBCC.join(', ')
  }
}))

// Setup draft management composable, keyed per conversation and message type.
const {
  htmlContent,
  textContent,
  isLoading: isDraftLoading,
  captureDraft,
  completeSend,
  loadedAttachments,
  recipients,
  saveState: draftSaveState,
  retrySave: retryDraftSave
} = useDraftManager(currentConversationUUID, messageType, mediaFiles, recipientDefaults)

// Rest of existing state
const isEditorFullscreen = ref(false)
const isSending = ref(false)

const recipientModel = (field) => computed({
  get: () => recipients.value[field],
  set: (value) => { recipients.value[field] = value }
})
const to = recipientModel('to')
const cc = recipientModel('cc')
const bcc = recipientModel('bcc')
const showBcc = ref(false)
const emailErrors = ref([])
const replyBoxContentRef = ref(null)
const fullscreenContentRef = ref(null)
const activeContentRef = () =>
  isEditorFullscreen.value ? fullscreenContentRef.value : replyBoxContentRef.value
const showContactEmailWarning = ref(false)

const deferredStatus = ref(null)
const mentions = ref([])

const setMessageTypeFromPalette = (type) => {
  if (!isAllowedMessageType(type)) return
  messageType.value = type
}

const focusFromPalette = () => {
  // The cramped layout renders no editor until the fullscreen dialog opens.
  if (isCramped.value && !isEditorFullscreen.value) {
    isEditorFullscreen.value = true
    nextTick(() => fullscreenContentRef.value?.focus())
    return
  }
  activeContentRef()?.focus()
}

onMounted(() => {
  emitter.on(EMITTER_EVENTS.REPLY_BOX_SET_TYPE, setMessageTypeFromPalette)
  emitter.on(EMITTER_EVENTS.REPLY_BOX_FOCUS, focusFromPalette)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REPLY_BOX_SET_TYPE, setMessageTypeFromPalette)
  emitter.off(EMITTER_EVENTS.REPLY_BOX_FOCUS, focusFromPalette)
})

/**
 * Returns true if the editor has text content.
 */
const hasTextContent = computed(() => {
  return textContent.value.trim().length > 0
})

const draftPreview = computed(() => textContent.value.trim())

const attachmentCount = computed(() => mediaFiles.value.length + uploadingFiles.value.length)

const { newerReply, acknowledgeReply } = useReplyAwareness({
  uuid: currentConversationUUID,
  messageType,
  hasDraft: computed(() => hasTextContent.value || hasInlineImage(htmlContent.value) || mediaFiles.value.length > 0),
  messages: computed(() => conversationStore.conversationMessages),
  ready: computed(() => {
    void conversationStore.messages.version
    return conversationStore.messages.data.hasConversation(currentConversationUUID.value)
  }),
  userID: computed(() => userStore.userID)
})

const processSend = async (skipContactEmailCheck = false, statusToSet = null) => {
  if (isSending.value || isDraftLoading.value || !conversationStore.current.uuid) return
  let hasMessageSendingErrored = false
  isEditorFullscreen.value = false

  const html = htmlContent.value
  if (hasPendingInlineUpload(html)) return
  const hasContent = hasTextContent.value || hasInlineImage(html) || mediaFiles.value.length > 0
  const convUUID = conversationStore.current.uuid
  const isPrivate = messageType.value === 'private_note'

  if ((isPrivate && !canSendPrivateNote.value) || (!isPrivate && !canSendReply.value)) return

  if (!isPrivate && conversationStore.current.inbox_channel === 'email') {
    // Require at least one recipient in `to`.
    if (!to.value.trim()) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: t('replyBox.toRequired')
      })
      return
    }

    // Warn if the contact's email is not in any recipient field.
    if (!skipContactEmailCheck) {
      const contactEmail = conversationStore.current.correspondent?.email?.toLowerCase()
      if (contactEmail) {
        const allRecipients = [to.value, cc.value, bcc.value].join(',').toLowerCase()
        const intendedRecipients = [contactEmail, ...conversationStore.currentTo.map(email => email.toLowerCase())]
        if (
          !allRecipients
            .split(',')
            .map((e) => e.trim())
            .some((email) => intendedRecipients.includes(email))
        ) {
          deferredStatus.value = statusToSet
          showContactEmailWarning.value = true
          return
        }
      }
    }
  }
  let tempUUID = null
  let draftSnapshot = null

  // Add pending message to cache for instant display.
  if (hasContent) {
    draftSnapshot = captureDraft()
    const savedContent = htmlContent.value
    const author = {
      id: userStore.userID,
      first_name: userStore.firstName,
      last_name: userStore.lastName,
      avatar_url: userStore.avatar,
      type: 'agent'
    }
    const parsedTo =
      !isPrivate && to.value
        ? to.value
            .split(',')
            .map((e) => e.trim())
            .filter(Boolean)
        : []
    const parsedCC =
      !isPrivate && cc.value
        ? cc.value
            .split(',')
            .map((e) => e.trim())
            .filter(Boolean)
        : []
    const parsedBCC =
      !isPrivate && bcc.value
        ? bcc.value
            .split(',')
            .map((e) => e.trim())
            .filter(Boolean)
        : []
    const meta = {}
    if (parsedTo.length) meta.to = parsedTo
    if (parsedCC.length) meta.cc = parsedCC
    if (parsedBCC.length) meta.bcc = parsedBCC

    tempUUID = conversationStore.addPendingMessage(
      convUUID,
      savedContent,
      isPrivate,
      author,
      mediaFiles.value,
      textContent.value,
      meta
    )

    // Keep the draft until the server acknowledges durable storage.
    try {
      isSending.value = true
      const response = await api.sendMessage(convUUID, {
        sender_type: UserTypeAgent,
        private: isPrivate,
        message: savedContent,
        attachments: mediaFiles.value.map((file) => file.id),
        mentions: isPrivate ? mentions.value : [],
        cc: parsedCC,
        bcc: parsedBCC,
        to: parsedTo,
        echo_id: isPrivate ? '' : tempUUID
      })

      // The HTTP acknowledgement contains the durable message; WS may be offline.
      if (response?.data?.data?.uuid) {
        conversationStore.replacePendingMessage(convUUID, tempUUID, response.data.data)
      }
    } catch (error) {
      hasMessageSendingErrored = true
      // The original draft remains intact, even after navigating to another thread.
      conversationStore.removePendingMessage(convUUID, tempUUID)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  // Clear state on success.
  if (!hasMessageSendingErrored) {
    if (draftSnapshot && completeSend(draftSnapshot)) {
      emailErrors.value = []
      mentions.value = []
    }
    if (statusToSet) await conversationStore.updateStatus(statusToSet, convUUID)
  }
  isSending.value = false
}

const processSendAndSetStatus = (status) => processSend(false, status)

/**
 * Watch for loaded attachments from draft and restore them to mediaFiles.
 */
watch(
  loadedAttachments,
  (attachments) => {
    setMediaFiles([...attachments])
  },
  { deep: true }
)

watch(bcc, (value) => { showBcc.value = Boolean(value) })

// Media files are restored per draft by the draft manager; resetting here would race ahead of the save and drop them.
watch(
  () => conversationStore.current?.uuid,
  () => {
    setTimeout(() => {
      activeContentRef()?.focus()
    }, 100)
  }
)
</script>
