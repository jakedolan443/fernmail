import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { navIconMap } from '@main/constants/navIcons'
import { adminNavItems } from '@main/constants/navigation'
import { SECTIONS } from '../sections'

// Fernmail is reply-only. The command palette only exposes administrative
// creation flows; it cannot create a new outbound conversation or saved view.
export function useCreateCommands() {
  const router = useRouter()
  const { t } = useI18n()
  const adminCreates = () =>
    adminNavItems
      .flatMap((group) => group.children)
      .filter((item) => item.createRouteName)
      .map((item) => ({
        id: `create.${item.createRouteName}`,
        label: t(router.resolve({ name: item.createRouteName }).meta.titleKey),
        keywords: [t('globals.messages.create'), t('globals.terms.admin')],
        section: SECTIONS.CREATE,
        icon: navIconMap[item.icon],
        permission: item.permission,
        run: () => router.push({ name: item.createRouteName })
      }))
  return computed(() => adminCreates())
}
