// Separate development entry: never imported by the production application.
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import mitt from 'mitt'
import { initI18n } from '../i18n'
import messages from '../../../../../i18n/en-US.json'
import '@shared-ui/assets/styles/main.scss'
import '@shared-ui/utils/string.js'
import MailPreview from './MailPreview.vue'

if (!import.meta.env.DEV) throw new Error('The mock mailbox is development-only.')
const router = createRouter({
  history: createMemoryHistory(),
  routes: [
    { path: '/', redirect: '/inboxes/all/conversation/albert' },
    { path: '/inboxes/:type', name: 'inbox', component: {} },
    { path: '/inboxes/:type/conversation/:uuid', name: 'inbox-conversation', component: {} }
  ]
})
const app = createApp(MailPreview)
app.config.globalProperties.emitter = mitt()
app.use(createPinia())
app.use(initI18n({ legacy: false, locale: 'en-US', messages: { 'en-US': messages } }))
app.use(router)
await router.push('/inboxes/all/conversation/albert')
app.mount('#app')
