<script setup lang="ts">
withDefaults(defineProps<{ title: string; description?: string }>(), {})
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div class="space-y-1">
      <h1 class="font-display text-2xl font-semibold tracking-tight text-foreground">{{ title }}</h1>
      <p v-if="description" class="text-sm text-muted-foreground">{{ description }}</p>
    </div>
    <div class="flex items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
