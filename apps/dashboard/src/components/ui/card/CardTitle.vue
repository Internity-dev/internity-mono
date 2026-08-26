<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    class?: HTMLAttributes['class']
    as?: 'div' | 'h1' | 'h2' | 'h3' | 'h4'
  }>(),
  { as: 'div' },
)
</script>

<template>
  <component
    :is="props.as"
    data-slot="card-title"
    :class="cn('text-base leading-snug font-medium group-data-[size=sm]/card:text-sm cn-font-heading', props.class)"
  >
    <slot />
  </component>
</template>
