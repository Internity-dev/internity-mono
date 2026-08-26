<script setup lang="ts">
import { computed } from 'vue'
import { StarIcon } from '@lucide/vue'

const props = defineProps<{ modelValue: number | undefined }>()
const emit = defineEmits<{ 'update:modelValue': [value: number] }>()
const value = computed(() => props.modelValue ?? 0)
</script>

<template>
  <div class="flex items-center gap-1" role="radiogroup" aria-label="Rating">
    <button
      v-for="n in 5"
      :key="n"
      type="button"
      role="radio"
      :aria-checked="value === n"
      :aria-label="`${n} out of 5`"
      class="rounded-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
      @click="emit('update:modelValue', n)"
    >
      <StarIcon :class="['size-6', n <= value ? 'fill-warning text-warning' : 'text-muted-foreground']" />
    </button>
  </div>
</template>
