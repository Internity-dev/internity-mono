<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { CompassIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
</script>

<template>
  <div class="flex min-h-screen flex-col items-center justify-center gap-4 p-6 text-center">
    <div class="flex size-14 items-center justify-center rounded-full bg-muted text-muted-foreground">
      <CompassIcon class="size-7" />
    </div>
    <div class="space-y-1">
      <h1 class="font-display text-2xl font-semibold tracking-tight">Page not found</h1>
      <p class="text-sm text-muted-foreground">The page you're looking for doesn't exist or was moved.</p>
    </div>
    <Button as-child>
      <RouterLink to="/dashboard">Back to dashboard</RouterLink>
    </Button>
  </div>
</template>
