/**
 * Scroll-triggered reveal: returns a template ref to attach to an element and
 * a `visible` boolean that flips true once the element enters the viewport
 * (IntersectionObserver, one-shot). Pair with the `.reveal` / `.reveal-visible`
 * classes in main.css for a fade + slide-up transition, matching the
 * scroll-reveal feel used across the reference site's sections.
 */
export function useReveal(threshold = 0.2) {
  const target = ref<HTMLElement | null>(null)
  const visible = ref(false)

  onMounted(() => {
    if (!target.value) return
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting) {
          visible.value = true
          observer.disconnect()
        }
      },
      { threshold },
    )
    observer.observe(target.value)
  })

  return { target, visible }
}
