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
    variant="outline"
    class="ml-auto shrink-0 border-primary/35 bg-primary/15 px-1.5 py-0 font-medium tabular-nums text-primary"
    :aria-label="ariaLabel"
  >
    {{ count > 99 ? '99+' : String(count) }}
  </Badge>
</template>
