<template>
  <div class="flex min-h-20 w-full flex-col gap-1 px-2 py-2">
    <span class="min-w-0 w-full truncate text-sm font-semibold" :title="userStore.getFullName">
      {{ userStore.getFullName }}
    </span>
    <div class="flex w-full items-center justify-end gap-1">
      <BrowserNotifications />
      <Button
        v-if="isSettings || userStore.can('general_settings:manage')"
        variant="ghost"
        size="icon"
        class="h-9 w-9 shrink-0"
        :class="
          isSettings
            ? 'text-success hover:text-success'
            : 'text-muted-foreground hover:text-foreground'
        "
        :aria-label="isSettings ? 'Back to mailbox' : 'Settings'"
        :title="isSettings ? 'Back to mailbox' : 'Settings'"
        @click="router.push(isSettings ? '/inboxes/all' : '/admin/general')"
      >
        <ArrowLeft v-if="isSettings" class="h-4 w-4" aria-hidden="true" />
        <Settings v-else class="h-4 w-4" aria-hidden="true" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        class="h-9 w-9 shrink-0 text-muted-foreground hover:text-foreground"
        :aria-label="mode === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'"
        :title="mode === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'"
        @click="mode = mode === 'dark' ? 'light' : 'dark'"
      >
        <Sun v-if="mode === 'dark'" class="h-4 w-4" aria-hidden="true" />
        <Moon v-else class="h-4 w-4" aria-hidden="true" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        class="h-9 w-9 shrink-0 text-muted-foreground hover:text-foreground"
        :aria-label="t('navigation.logout')"
        :title="t('navigation.logout')"
        @click="logout"
      >
        <LogOut class="h-4 w-4" aria-hidden="true" />
      </Button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useColorMode } from '@vueuse/core'
import { useRoute, useRouter } from 'vue-router'
import { Settings, ArrowLeft, LogOut, Sun, Moon } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useUserStore } from '@main/stores/user'
import { useLogout } from '@main/composables/useLogout'
import BrowserNotifications from './BrowserNotifications.vue'

const mode = useColorMode()
const userStore = useUserStore()
const router = useRouter()
const route = useRoute()
const isSettings = computed(() => route.path === '/admin' || route.path.startsWith('/admin/'))
const { t } = useI18n()
const logout = useLogout()
</script>
