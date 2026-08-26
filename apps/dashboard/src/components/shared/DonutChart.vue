<script setup lang="ts">
import { computed } from 'vue'
import { Doughnut } from 'vue-chartjs'
import { Chart as ChartJS, ArcElement, Tooltip, Legend } from 'chart.js'
import { useChartColors } from '@/composables/useChartColors'
import { Skeleton } from '@/components/ui/skeleton'

ChartJS.register(ArcElement, Tooltip, Legend)

type ChartColor = 'chart1' | 'chart2' | 'chart3' | 'chart4' | 'chart5'

const props = defineProps<{
  data: { label: string; value: number; color: ChartColor }[]
  // While the underlying query is still pending, the caller's counts default
  // to 0 (see DashboardView's `?? 0`) — without this flag that reads to us as
  // "confirmed empty" and we'd render "No data yet" during every load.
  isLoading?: boolean
}>()

const colors = useChartColors()
const total = computed(() => props.data.reduce((sum, d) => sum + d.value, 0))

const chartData = computed(() => ({
  labels: props.data.map((d) => d.label),
  datasets: [
    {
      data: props.data.map((d) => d.value),
      backgroundColor: props.data.map((d) => colors.value[`--${d.color}`] || '#98a1b3'),
      borderColor: colors.value['--popover'] || '#ffffff',
      borderWidth: 2,
      hoverOffset: 6,
    },
  ],
}))

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: '65%',
  plugins: {
    legend: {
      position: 'right' as const,
      labels: {
        boxWidth: 8,
        boxHeight: 8,
        usePointStyle: true,
        pointStyle: 'circle' as const,
        padding: 12,
        color: colors.value['--muted-foreground'] || '#717a8f',
        font: { size: 12 },
      },
    },
    tooltip: {
      backgroundColor: colors.value['--popover'] || '#ffffff',
      titleColor: colors.value['--popover-foreground'] || '#171b26',
      bodyColor: colors.value['--popover-foreground'] || '#171b26',
      borderColor: colors.value['--border'] || '#dfe3ea',
      borderWidth: 1,
      padding: 8,
      cornerRadius: 8,
      displayColors: false,
    },
  },
}))
</script>

<template>
  <div v-if="isLoading" class="flex h-44 items-center gap-6 px-2">
    <Skeleton class="size-32 shrink-0 rounded-full" />
    <div class="space-y-2">
      <Skeleton class="h-3 w-24" />
      <Skeleton class="h-3 w-20" />
      <Skeleton class="h-3 w-24" />
    </div>
  </div>
  <p v-else-if="total === 0" class="py-6 text-center text-sm text-muted-foreground">No data yet</p>
  <div v-else class="h-44">
    <Doughnut :data="chartData" :options="chartOptions" />
  </div>
</template>
