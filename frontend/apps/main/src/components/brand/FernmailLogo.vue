<script setup>
defineProps({ name: { type: String, default: 'Fernmail' } })
</script>

<template>
  <span class="inline-flex items-center gap-2 font-semibold tracking-tight">
    <img src="/images/fern.svg" alt="" class="h-7 w-7 shrink-0" />
    <span class="truncate">{{ name }}</span>
  </span>
</template>
