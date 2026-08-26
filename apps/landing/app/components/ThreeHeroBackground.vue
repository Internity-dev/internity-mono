<script setup lang="ts">
/**
 * Subtle animated 3D background for the hero: a handful of low-poly
 * wireframe icosahedra in the brand's blue/teal, drifting and rotating
 * slowly at different depths. Purely decorative (aria-hidden, pointer-events
 * off), sits behind the hero text and the mockup card.
 */
const canvas = ref<HTMLCanvasElement | null>(null)
let frameId = 0
let cleanup: (() => void) | undefined

onMounted(async () => {
  if (!canvas.value) return
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return

  const THREE = await import('three')

  const scene = new THREE.Scene()
  const camera = new THREE.PerspectiveCamera(50, 1, 0.1, 100)
  camera.position.z = 18

  const renderer = new THREE.WebGLRenderer({ canvas: canvas.value, alpha: true, antialias: true })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))

  const colors = [0x1a43d6, 0x2e63f5, 0x059271]
  const shapes = colors.map((color, i) => {
    const geometry = new THREE.IcosahedronGeometry(2 + i * 0.6, 0)
    const material = new THREE.MeshBasicMaterial({ color, wireframe: true, transparent: true, opacity: 0.35 - i * 0.06 })
    const mesh = new THREE.Mesh(geometry, material)
    mesh.position.set(-4 + i * 5, i % 2 === 0 ? 2 : -3, -i * 3)
    scene.add(mesh)
    return mesh
  })

  function resize() {
    const el = canvas.value
    if (!el) return
    const { clientWidth, clientHeight } = el.parentElement ?? el
    camera.aspect = clientWidth / clientHeight
    camera.updateProjectionMatrix()
    renderer.setSize(clientWidth, clientHeight)
  }
  resize()
  window.addEventListener('resize', resize)

  function animate() {
    shapes.forEach((mesh, i) => {
      mesh.rotation.x += 0.0015 + i * 0.0004
      mesh.rotation.y += 0.002 + i * 0.0003
    })
    renderer.render(scene, camera)
    frameId = requestAnimationFrame(animate)
  }
  animate()

  cleanup = () => {
    window.removeEventListener('resize', resize)
    cancelAnimationFrame(frameId)
    shapes.forEach((mesh) => {
      mesh.geometry.dispose()
      ;(mesh.material as InstanceType<typeof THREE.Material>).dispose()
    })
    renderer.dispose()
  }
})

onUnmounted(() => cleanup?.())
</script>

<template>
  <canvas ref="canvas" aria-hidden="true" class="pointer-events-none absolute inset-0 h-full w-full" />
</template>
