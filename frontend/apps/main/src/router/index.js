import { createRouter, createWebHistory } from 'vue-router'
import { useAppSettingsStore } from '@main/stores/appSettings'
import { getI18n } from '@main/i18n'
import { abortRouteScope } from '@main/api'

const routes = [
  {
    path: '/',
    component: () => import('@main/OuterApp.vue'),
    children: [
      {
        path: '',
        name: 'login',
        component: () => import('@main/views/auth/UserLoginView.vue'),
        meta: { titleKey: 'auth.signInButton' }
      },
      {
        path: 'reset-password',
        name: 'reset-password',
        component: () => import('@main/views/auth/ResetPasswordView.vue'),
        meta: { titleKey: 'auth.resetPassword' }
      },
      {
        path: 'set-password',
        name: 'set-password',
        component: () => import('@main/views/auth/SetPasswordView.vue'),
        meta: { titleKey: 'auth.setNewPassword' }
      }
    ]
  },
  {
    path: '/',
    component: () => import('@main/App.vue'),
    children: [
      {
        path: '/inboxes/mailbox/:inboxID',
        component: () => import('@main/layouts/inbox/InboxLayout.vue'),
        meta: { titleKey: 'globals.terms.inbox', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'mailbox-inbox',
            component: () => import('@main/views/inbox/InboxView.vue'),
            children: [
              {
                path: 'conversation/:uuid',
                name: 'mailbox-inbox-conversation',
                component: () => import('@main/views/conversation/ConversationDetailView.vue'),
                props: true
              }
            ]
          }
        ]
      },
      {
        path: '/inboxes/views/:viewID',
        name: 'views',
        props: true,
        component: () => import('@main/layouts/inbox/InboxLayout.vue'),
        meta: { titleKey: 'globals.terms.view', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'view-inbox',
            component: () => import('@main/views/inbox/InboxView.vue'),
            meta: { titleKey: 'globals.terms.view' },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'view-inbox-conversation',
                component: () => import('@main/views/conversation/ConversationDetailView.vue'),
                props: true,
                meta: { titleKey: 'globals.terms.view', hidePageHeader: true }
              }
            ]
          }
        ]
      },
      {
        path: 'inboxes/search',
        name: 'search',
        component: () => import('@main/views/search/SearchView.vue'),
        meta: { titleKey: 'globals.terms.search', hidePageHeader: true }
      },
      {
        path: '/inboxes/:type(all|mentioned)?',
        name: 'inboxes',
        redirect: '/inboxes/all',
        component: () => import('@main/layouts/inbox/InboxLayout.vue'),
        props: true,
        meta: { titleKey: 'globals.terms.inbox', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'inbox',
            component: () => import('@main/views/inbox/InboxView.vue'),
            meta: {
              titleKey: 'globals.terms.inbox',
              typeKey: (route) => {
                if (route.params.type === 'mentioned') return 'conversation.mentions'
                if (route.params.type === 'all') return 'globals.messages.all'
                return ''
              }
            },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'inbox-conversation',
                component: () => import('@main/views/conversation/ConversationDetailView.vue'),
                props: true,
                meta: {
                  titleKey: 'globals.terms.inbox',
                  typeKey: (route) => {
                    if (route.params.type === 'mentioned') return 'conversation.mentions'
                    if (route.params.type === 'all') return 'globals.messages.all'
                    return ''
                  },
                  hidePageHeader: true
                }
              }
            ]
          }
        ]
      },

      {
        path: '/admin',
        name: 'admin',
        component: () => import('@main/layouts/admin/AdminLayout.vue'),
        meta: { titleKey: 'globals.terms.admin' },
        children: [
          {
            path: 'general',
            name: 'general',
            component: () => import('@main/views/admin/general/General.vue'),
            meta: { titleKey: 'globals.terms.general' }
          },
          {
            path: 'resources',
            name: 'system-resources',
            component: () => import('@main/views/admin/resources/ResourceMonitor.vue'),
            meta: { titleKey: 'admin.systemResources.title' }
          },

          {
            path: 'inboxes',
            component: () => import('@main/views/admin/inbox/InboxView.vue'),
            meta: { titleKey: 'globals.terms.inbox', titleCount: 2 },
            children: [
              {
                path: '',
                name: 'inbox-list',
                component: () => import('@main/views/admin/inbox/InboxList.vue')
              },
              {
                path: 'new',
                name: 'new-inbox',
                component: () => import('@main/views/admin/inbox/NewInbox.vue'),
                meta: { titleKey: 'inbox.newInbox' }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-inbox',
                component: () => import('@main/views/admin/inbox/EditInbox.vue'),
                meta: { titleKey: 'inbox.edit' }
              }
            ]
          },

          {
            path: 'templates',
            component: () => import('@main/views/admin/templates/Templates.vue'),
            meta: { titleKey: 'globals.terms.template', titleCount: 2 },
            children: [
              {
                path: '',
                name: 'template-list',
                component: () => import('@main/views/admin/templates/TemplateList.vue')
              },
              {
                path: ':id/edit',
                name: 'edit-template',
                props: true,
                component: () => import('@main/views/admin/templates/CreateEditTemplate.vue'),
                meta: { titleKey: 'template.edit' }
              },
              {
                path: 'new',
                name: 'new-template',
                props: true,
                component: () => import('@main/views/admin/templates/CreateEditTemplate.vue'),
                meta: { titleKey: 'template.new' }
              }
            ]
          },
          {
            path: 'sso',
            component: () => import('@main/views/admin/oidc/OIDC.vue'),
            name: 'sso',
            meta: { titleKey: 'globals.terms.sso' },
            children: [
              {
                path: '',
                name: 'sso-list',
                component: () => import('@main/views/admin/oidc/OIDCList.vue')
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-sso',
                component: () => import('@main/views/admin/oidc/CreateEditOIDC.vue'),
                meta: { titleKey: 'oidc.edit' }
              },
              {
                path: 'new',
                name: 'new-sso',
                component: () => import('@main/views/admin/oidc/CreateEditOIDC.vue'),
                meta: { titleKey: 'oidc.new' }
              }
            ]
          },
          {
            path: 'webhooks',
            component: () => import('@main/views/admin/webhooks/Webhooks.vue'),
            name: 'webhooks',
            meta: { titleKey: 'globals.terms.webhook', titleCount: 2 },
            children: [
              {
                path: '',
                name: 'webhook-list',
                component: () => import('@main/views/admin/webhooks/WebhookList.vue')
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-webhook',
                component: () => import('@main/views/admin/webhooks/CreateEditWebhook.vue'),
                meta: { titleKey: 'webhook.edit' }
              },
              {
                path: 'new',
                name: 'new-webhook',
                component: () => import('@main/views/admin/webhooks/CreateEditWebhook.vue'),
                meta: { titleKey: 'webhook.new' }
              }
            ]
          },

          {
            path: 'conversations',
            meta: { titleKey: 'globals.terms.conversation', titleCount: 2 },
            children: [
              {
                path: 'statuses',
                component: () => import('@main/views/admin/status/StatusView.vue'),
                meta: { titleKey: 'globals.terms.status', titleCount: 2 }
              },

              {
                path: 'shared-views',
                component: () => import('@main/views/admin/shared-views/SharedViews.vue'),
                meta: { titleKey: 'globals.terms.sharedView', titleCount: 2 },
                children: [
                  {
                    path: '',
                    name: 'shared-view-list',
                    component: () => import('@main/views/admin/shared-views/SharedViewList.vue')
                  },
                  {
                    path: 'new',
                    name: 'new-shared-view',
                    component: () => import('@main/views/admin/shared-views/CreateSharedView.vue'),
                    meta: { titleKey: 'sharedView.new' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-shared-view',
                    component: () => import('@main/views/admin/shared-views/EditSharedView.vue'),
                    meta: { titleKey: 'sharedView.editSharedView' }
                  }
                ]
              }
            ]
          }
        ]
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: () => {
      return '/inboxes/all'
    }
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: routes
})

router.beforeEach((to, from, next) => {
  // Cancel in-flight requests.
  if (to.fullPath !== from.fullPath) {
    abortRouteScope()
  }

  const appSettingsStore = useAppSettingsStore()
  const siteName = appSettingsStore.settings?.['app.site_name'] || 'Fernmail'
  const i18n = getI18n()
  const typeKey = typeof to.meta?.typeKey === 'function' ? to.meta.typeKey(to) : ''
  const titleKey = typeKey || to.meta?.titleKey
  const pageTitle = titleKey && i18n ? i18n.global.t(titleKey, to.meta?.titleCount || 1) : ''
  document.title = `${pageTitle} - ${siteName}`
  next()
})

export default router
