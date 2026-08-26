<script setup lang="ts">
import { ref, watch } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import { SearchIcon } from '@lucide/vue'
import { Input } from '@/components/ui/input'

const props = defineProps<{
  modelValue: string
  placeholder?: string
}>()

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const local = ref(props.modelValue)
watch(
  () => props.modelValue,
  (v) => {
    if (v !== local.value) local.value = v
  },
)

const emitDebounced = useDebounceFn((v: string) => emit('update:modelValue', v), 300)
watch(local, (v) => emitDebounced(v))
</script>

<template>
  <div class="flex flex-wrap items-center gap-3">
    <div class="relative w-full max-w-xs">
      <SearchIcon class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
      <Input v-model="local" :placeholder="placeholder ?? 'Search…'" class="pl-8" />
    </div>
    <slot />
  </div>
</template>
