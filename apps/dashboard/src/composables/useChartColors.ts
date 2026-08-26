import { onMounted, onUnmounted, ref } from 'vue'

// Canvas-based charts (chart.js) need resolved color strings, not CSS
// `var(...)` references — so the design tokens' --chart1..5 custom
// properties are read via getComputedStyle and re-read whenever the .dark
// class flips, instead of being passed through as raw var() strings.
const CHART_VARS = ['--chart1', '--chart2', '--chart3', '--chart4', '--chart5', '--muted-foreground', '--popover', '--popover-foreground', '--border'] as const

export function useChartColors() {
  const colors = ref<Record<string, string>>({})

  function resolve() {
    const styles = getComputedStyle(document.documentElement)
    const next: Record<string, string> = {}
    for (const name of CHART_VARS) next[name] = styles.getPropertyValue(name).trim()
    colors.value = next
  }

  let observer: MutationObserver | undefined
  onMounted(() => {
    resolve()
    observer = new MutationObserver(resolve)
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  })
  onUnmounted(() => observer?.disconnect())

  return colors
}
