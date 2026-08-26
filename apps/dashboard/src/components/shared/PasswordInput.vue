<script setup lang="ts">
import { ref, type HTMLAttributes } from 'vue'
import { useVModel } from '@vueuse/core'
import { EyeIcon, EyeOffIcon } from '@lucide/vue'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  defaultValue?: string
  modelValue?: string
  class?: HTMLAttributes['class']
}>()

const emits = defineEmits<{
  (e: 'update:modelValue', payload: string | number): void
}>()

const modelValue = useVModel(props, 'modelValue', emits, {
  passive: true,
  defaultValue: props.defaultValue,
})

const visible = ref(false)
</script>

<template>
  <div class="relative">
    <Input
      v-model="modelValue"
      v-bind="$attrs"
      :type="visible ? 'text' : 'password'"
      :class="cn('pr-9', props.class)"
    />
    <button
      type="button"
      :aria-label="visible ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'"
      :aria-pressed="visible"
      class="absolute top-1/2 right-2.5 -translate-y-1/2 text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 rounded-xs"
      @click="visible = !visible"
    >
      <EyeOffIcon v-if="visible" class="size-4" />
      <EyeIcon v-else class="size-4" />
    </button>
  </div>
</template>
