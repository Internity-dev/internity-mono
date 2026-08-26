<script setup lang="ts" generic="T extends object">
import { ArrowDownIcon, ArrowUpIcon } from '@lucide/vue'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import EmptyState from '@/components/shared/EmptyState.vue'

export interface Column {
  key: string
  label: string
  sortable?: boolean
  class?: string
}

const props = defineProps<{
  columns: Column[]
  rows: T[]
  isLoading?: boolean
  sort?: string
  order?: 'asc' | 'desc'
  emptyTitle?: string
  emptyDescription?: string
}>()

defineEmits<{ sort: [key: string] }>()

defineSlots<{
  [key: `cell-${string}`]: (props: { row: T }) => unknown
}>()

// The default (un-slotted) cell render just stringifies whatever's at this
// key — callers needing real formatting provide a `#cell-{key}` slot instead.
function cellValue(row: T, key: string): unknown {
  return (row as Record<string, unknown>)[key]
}
</script>

<template>
  <div class="overflow-x-auto rounded-lg border">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead
            v-for="col in columns"
            :key="col.key"
            :class="[col.class, col.sortable && 'cursor-pointer select-none']"
            @click="col.sortable && $emit('sort', col.key)"
          >
            <span class="inline-flex items-center gap-1">
              {{ col.label }}
              <ArrowUpIcon v-if="col.sortable && sort === col.key && order === 'asc'" class="size-3" />
              <ArrowDownIcon v-if="col.sortable && sort === col.key && order === 'desc'" class="size-3" />
            </span>
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <template v-if="isLoading">
          <TableRow v-for="i in 5" :key="i">
            <TableCell v-for="col in columns" :key="col.key">
              <Skeleton class="h-4 w-full" />
            </TableCell>
          </TableRow>
        </template>
        <TableRow v-else-if="rows.length === 0">
          <TableCell :colspan="columns.length">
            <EmptyState :title="emptyTitle ?? 'No results'" :description="emptyDescription" />
          </TableCell>
        </TableRow>
        <TableRow v-for="(row, i) in rows" v-else :key="i">
          <TableCell v-for="col in columns" :key="col.key">
            <slot :name="`cell-${col.key}`" :row="row">{{ cellValue(row, col.key) }}</slot>
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>
