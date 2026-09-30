import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { initI18n } from './i18n'
import { useAppSettingsStore } from './stores/appSettings'
import router from './router'
import mitt from 'mitt'
import api from './api'
import '@shared-ui/assets/styles/main.scss'
import '@shared-ui/utils/string.js'
import Root from './Root.vue'

const setFavicon = (url) => {
  const configured = new URL(url, window.location.origin)
  // Keep the versioned bundled icon for default favicon settings.
  if (
    ['/favicon.ico', '/favicon.svg'].includes(configured.pathname) &&
    [window.location.origin, 'http://localhost:9000'].includes(configured.origin)
  )
    return
  let link = document.querySelector('#app-favicon') || document.createElement('link')
  link.rel = 'icon'
  link.removeAttribute('type')
  document.head.appendChild(link)
  link.href = url
}

async function initApp() {
  const config = (await api.getConfig()).data.data
  const emitter = mitt()
  const lang = config['app.lang'] || 'en-US'
  const [langMessages, fallbackMessages] = await Promise.all([
    api.getLanguage(lang),
    lang === 'en-US' ? Promise.resolve(null) : api.getLanguage('en-US')
  ])

  // Set favicon.
  if (config['app.favicon_url']) setFavicon(config['app.favicon_url'])

  // Initialize i18n.
  const i18nConfig = {
    legacy: false,
    locale: lang,
    fallbackLocale: 'en-US',
    messages: {
      'en-US': fallbackMessages?.data || langMessages.data,
      [lang]: langMessages.data
    }
  }

  const i18n = initI18n(i18nConfig)
  const app = createApp(Root)
  const pinia = createPinia()
  app.use(pinia)

  // Fetch and store app settings in store (after pinia is initialized)
  const settingsStore = useAppSettingsStore()

  // Store the public config in the store
  settingsStore.setPublicConfig(config)

  try {
    await settingsStore.fetchSettings('general')
  } catch (error) {
    // Pass
  }

  // Add emitter to global properties.
  app.config.globalProperties.emitter = emitter

  app.use(router)
  app.use(i18n)
  app.mount('#app')
}

initApp().catch((error) => {
  console.error('Error initializing app: ', error)
})
