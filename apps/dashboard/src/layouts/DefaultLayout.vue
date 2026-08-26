<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { MenuIcon, XIcon, LogOutIcon, MoonIcon, SunIcon, HelpCircleIcon, SearchIcon } from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'
import { navSectionsForRole } from '@/lib/nav'
import { useTour } from '@/composables/useTour'
import { showMenuHintIfFirstVisit, markHintsSeenFor } from '@/composables/useMenuHints'
import { tourStepsForRole, coreHintPathsForRole } from '@/tours'
import { avatarUrl } from '@/lib/avatar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator, DropdownMenuLabel } from '@/components/ui/dropdown-menu'
import NotificationBell from '@/components/shared/NotificationBell.vue'
import { useDark, useToggle } from '@vueuse/core'
import logo from '@/assets/logo.png'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const sections = computed(() => (auth.user ? navSectionsForRole(auth.user.role) : []))
const sidebarOpen = ref(false)

// Sidebar quick-nav filter: narrows the nav items shown as you type, Enter
// jumps to the first match. Not a fake decoration — it's a real, if small,
// piece of functionality behind the search-bar look.
const navFilter = ref('')
const navSearchInput = ref<InstanceType<typeof Input> | null>(null)
const filteredSections = computed(() => {
  const q = navFilter.value.trim().toLowerCase()
  if (!q) return sections.value
  return sections.value
    .map((section) => ({ ...section, items: section.items.filter((item) => item.label.toLowerCase().includes(q)) }))
    .filter((section) => section.items.length > 0)
})
const shortcutHint = navigator.platform.includes('Mac') ? '⌘K' : 'Ctrl K'
function goToFirstMatch() {
  const first = filteredSections.value[0]?.items[0]
  if (first) {
    router.push(first.to)
    navFilter.value = ''
    sidebarOpen.value = false
  }
}
function onGlobalKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    navSearchInput.value?.$el?.focus()
  }
}
onMounted(() => window.addEventListener('keydown', onGlobalKeydown))
onUnmounted(() => window.removeEventListener('keydown', onGlobalKeydown))

// Defaults to light regardless of OS preference — a school admin tool reads
// friendlier light-first. Once the user toggles it manually, that choice is
// remembered (useDark persists to localStorage under the hood). Storage key
// changed from vueuse's default so anyone who toggled dark earlier in this
// project's life (before light became the default) gets a clean reset
// instead of staying stuck on a stale "dark" value forever.
const isDark = useDark({ initialValue: 'light', storageKey: 'internity-color-scheme' })
const toggleDark = useToggle(isDark)

const initials = computed(() => {
  const name = auth.user?.name ?? ''
  return name
    .split(' ')
    .map((p) => p[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()
})
const avatarSrc = computed(() => avatarUrl(auth.user?.avatar_key))

// Guided tour: auto-starts once per role (see composables/useTour.ts), and
// can be replayed anytime from the user menu ("Replay tour" below).
function replayTour() {
  if (!auth.user) return
  useTour(auth.user.role, tourStepsForRole(auth.user.role)).start()
}

onMounted(() => {
  if (!auth.user) return
  // Let the sidebar/header finish rendering before targeting elements in it.
  requestAnimationFrame(() => {
    const tour = useTour(auth.user!.role, tourStepsForRole(auth.user!.role))
    // The core tour is about to spotlight these — don't immediately
    // re-explain them via a menu hint the moment the user clicks one.
    if (!tour.hasSeenTour()) markHintsSeenFor(coreHintPathsForRole(auth.user!.role))
    tour.startIfFirstVisit()
  })
})

// Progressive disclosure for everything the core tour doesn't cover: the
// first time the user actually lands on a menu, spotlight it once. See
// composables/useMenuHints.ts.
watch(
  () => route.path,
  (path) => {
    if (!auth.user) return
    requestAnimationFrame(() => showMenuHintIfFirstVisit(path))
  },
  { immediate: true },
)
</script>

<template>
  <div class="flex min-h-screen bg-background">
    <div
      v-if="sidebarOpen"
      class="fixed inset-0 z-40 bg-black/40 lg:hidden"
      @click="sidebarOpen = false"
    />

    <aside
      class="fixed inset-y-0 left-0 z-50 flex w-56 -translate-x-full flex-col border-r bg-sidebar transition-transform lg:translate-x-0"
      :class="{ 'translate-x-0': sidebarOpen }"
    >
      <div class="flex h-14 shrink-0 items-center justify-between border-b px-4">
        <RouterLink to="/dashboard" class="flex items-center gap-2 font-display text-lg font-semibold tracking-tight text-sidebar-primary">
          <img :src="logo" alt="" class="h-7 w-7" />
          Internity
        </RouterLink>
        <Button variant="ghost" size="icon" class="lg:hidden" aria-label="Close sidebar" @click="sidebarOpen = false">
          <XIcon class="size-5" />
        </Button>
      </div>

      <div class="shrink-0 border-b p-3">
        <div class="relative">
          <SearchIcon class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            ref="navSearchInput"
            v-model="navFilter"
            placeholder="Search"
            class="h-8 pr-12 pl-8 text-sm"
            @keydown.enter="goToFirstMatch"
          />
          <kbd
            v-if="!navFilter"
            class="pointer-events-none absolute top-1/2 right-2 -translate-y-1/2 rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground"
          >
            {{ shortcutHint }}
          </kbd>
        </div>
      </div>

      <nav class="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto p-3">
        <p v-if="navFilter && filteredSections.length === 0" class="px-2 py-4 text-center text-sm text-muted-foreground">
          No matches for "{{ navFilter }}"
        </p>
        <div v-for="section in filteredSections" :key="section.label ?? 'main'" class="space-y-1">
          <p v-if="section.label" class="px-2 pb-1.5 text-xs font-medium tracking-wide text-muted-foreground uppercase">
            {{ section.label }}
          </p>
          <RouterLink
            v-for="item in section.items"
            :key="item.to"
            :to="item.to"
            data-tour-nav
            :data-tour-nav-target="item.to"
            class="flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium transition-colors"
            :class="
              route.path.startsWith(item.to)
                ? 'bg-sidebar-primary text-sidebar-primary-foreground dark:bg-primary-400/15 dark:text-primary-300'
                : 'text-sidebar-foreground hover:bg-sidebar-accent'
            "
            @click="sidebarOpen = false"
          >
            <component :is="item.icon" class="size-4" />
            {{ item.label }}
          </RouterLink>
        </div>
      </nav>
    </aside>

    <div class="flex min-w-0 flex-1 flex-col lg:pl-56">
      <header class="sticky top-0 z-30 flex h-14 items-center justify-between gap-2 border-b bg-background/95 px-4 backdrop-blur">
        <Button variant="ghost" size="icon" class="lg:hidden" aria-label="Open sidebar" @click="sidebarOpen = true">
          <MenuIcon class="size-5" />
        </Button>
        <div class="flex-1" />
        <Button variant="ghost" size="icon" :aria-label="isDark ? 'Switch to light mode' : 'Switch to dark mode'" @click="toggleDark()">
          <SunIcon v-if="isDark" class="size-4" />
          <MoonIcon v-else class="size-4" />
        </Button>
        <NotificationBell />
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <button class="flex items-center gap-2 rounded-full" data-tour="user-menu">
              <Avatar class="size-8">
                <AvatarImage v-if="avatarSrc" :src="avatarSrc" :alt="auth.user?.name ?? ''" />
                <AvatarFallback>{{ initials }}</AvatarFallback>
              </Avatar>
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-56">
            <DropdownMenuLabel>
              <p class="font-medium">{{ auth.user?.name }}</p>
              <p class="text-xs font-normal text-muted-foreground capitalize">{{ auth.user?.role }}</p>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem as-child>
              <RouterLink to="/profile">Profile</RouterLink>
            </DropdownMenuItem>
            <DropdownMenuItem @click="replayTour()">
              <HelpCircleIcon class="size-4" />
              Replay tour
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" @click="auth.logout()">
              <LogOutIcon class="size-4" />
              Log out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </header>

      <main class="flex-1 space-y-6 p-4 sm:p-6">
        <RouterView />
      </main>
    </div>
  </div>
</template>
