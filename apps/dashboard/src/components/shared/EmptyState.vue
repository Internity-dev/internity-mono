<script setup lang="ts">
import type { Component } from 'vue'
import { InboxIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'

withDefaults(
  defineProps<{
    icon?: Component
    title: string
    description?: string
    actionLabel?: string
  }>(),
  { icon: undefined },
)

defineEmits<{ action: [] }>()
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-3 py-16 text-center">
    <div class="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
      <component :is="icon ?? InboxIcon" class="size-6" />
    </div>
    <div class="space-y-1">
      <p class="font-medium text-foreground">{{ title }}</p>
      <p v-if="description" class="max-w-sm text-sm text-muted-foreground">{{ description }}</p>
    </div>
    <Button v-if="actionLabel" size="sm" class="mt-2" @click="$emit('action')">{{ actionLabel }}</Button>
  </div>
</template>
