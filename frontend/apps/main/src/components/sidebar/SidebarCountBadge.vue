<script setup>
import { Badge } from '@shared-ui/components/ui/badge'

defineProps({
  count: { type: Number, default: 0 },
  ariaLabel: { type: String, default: '' }
})
</script>

<template>
  <Badge
    v-if="count > 0"
    class="ml-auto flex min-h-6 min-w-6 shrink-0 items-center justify-center rounded-full border-success bg-success px-2 py-0.5 text-xs font-semibold leading-none tabular-nums text-success-foreground shadow-sm max-md:min-h-7 max-md:min-w-7 max-md:px-2.5 max-md:text-sm"
    :aria-label="ariaLabel"
  >
    {{ count > 99 ? '99+' : String(count) }}
  </Badge>
</template>
