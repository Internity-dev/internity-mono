<script setup lang="ts">
import { computed } from 'vue'
import { Badge } from '@/components/ui/badge'

const props = withDefaults(
  defineProps<{
    tone?: 'success' | 'warning' | 'danger' | 'info' | 'neutral'
    label: string
  }>(),
  { tone: 'neutral' },
)

// One place mapping business-meaning -> color, per spec's "no duplicated
// components with the same function" — every status badge in the app (
// appliance/presence/journal/vacancy/news) goes through this same component.
const toneClass = computed(
  () =>
    ({
      success: 'bg-success/15 text-success border-success/30',
      warning: 'bg-warning/15 text-warning border-warning/30',
      danger: 'bg-danger/15 text-danger border-danger/30',
      info: 'bg-info/15 text-info border-info/30',
      neutral: 'bg-muted text-muted-foreground border-transparent',
    })[props.tone],
)
</script>

<template>
  <Badge variant="outline" :class="toneClass">{{ label }}</Badge>
</template>
