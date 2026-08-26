<template>
  <div class="flex min-h-screen flex-col bg-background text-foreground">
    <TheHeader />
    <main class="flex-1">
      <slot />
    </main>
    <TheFooter />
  </div>
</template>
