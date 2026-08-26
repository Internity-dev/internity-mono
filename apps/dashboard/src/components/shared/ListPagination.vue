<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeftIcon, ChevronRightIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'

const props = defineProps<{
  page: number
  limit: number
  total: number
}>()

const emit = defineEmits<{ 'update:page': [page: number] }>()

const lastPage = computed(() => Math.max(1, Math.ceil(props.total / props.limit)))
const rangeStart = computed(() => (props.total === 0 ? 0 : (props.page - 1) * props.limit + 1))
const rangeEnd = computed(() => Math.min(props.page * props.limit, props.total))
</script>

<template>
  <div v-if="total > 0" class="flex items-center justify-between gap-4 pt-3">
    <p class="text-sm text-muted-foreground">
      Showing <span class="font-medium text-foreground">{{ rangeStart }}–{{ rangeEnd }}</span> of
      <span class="font-medium text-foreground">{{ total }}</span>
    </p>
    <div class="flex items-center gap-2">
      <Button
        variant="outline"
        size="sm"
        :disabled="page <= 1"
        @click="emit('update:page', page - 1)"
      >
        <ChevronLeftIcon class="size-4" />
        Previous
      </Button>
      <span class="text-sm text-muted-foreground">Page {{ page }} of {{ lastPage }}</span>
      <Button
        variant="outline"
        size="sm"
        :disabled="page >= lastPage"
        @click="emit('update:page', page + 1)"
      >
        Next
        <ChevronRightIcon class="size-4" />
      </Button>
    </div>
  </div>
</template>
