<script setup lang="ts">
import { ref } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { useRoute, useRouter } from 'vue-router'
import { RouterLink } from 'vue-router'
import { toast } from 'vue-sonner'
import { MailIcon, LockIcon, Loader2Icon } from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'
import { errorMessage, retryAfterSeconds } from '@/lib/api-error'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import PasswordInput from '@/components/shared/PasswordInput.vue'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'

const schema = toTypedSchema(
  z.object({
    email: z.string().min(1, 'Email wajib diisi').email('Masukkan email yang valid'),
    password: z.string().min(1, 'Kata sandi wajib diisi'),
  }),
)

const { defineField, handleSubmit, errors } = useForm({ validationSchema: schema })
const [email, emailAttrs] = defineField('email')
const [password, passwordAttrs] = defineField('password')

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const isSubmitting = ref(false)
const serverError = ref<string | null>(null)

const onSubmit = handleSubmit(async (values) => {
  isSubmitting.value = true
  serverError.value = null
  try {
    await auth.login(values.email, values.password)
    toast.success('Selamat datang kembali')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    router.push(redirect)
  } catch (err: unknown) {
    const retryAfter = retryAfterSeconds(err)
    const fallback = retryAfter
      ? `Terlalu banyak percobaan. Coba lagi dalam sekitar ${retryAfter} detik.`
      : 'Email atau kata sandi salah.'
    serverError.value = errorMessage(err, fallback)
  } finally {
    isSubmitting.value = false
  }
})
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle as="h1">Selamat datang kembali</CardTitle>
      <CardDescription>Masuk untuk melanjutkan ke dashboard Anda.</CardDescription>
    </CardHeader>
    <CardContent>
      <Alert v-if="serverError" variant="destructive" class="mb-4">
        <AlertTitle>Gagal masuk</AlertTitle>
        <AlertDescription>{{ serverError }}</AlertDescription>
      </Alert>

      <form class="space-y-4" novalidate @submit="onSubmit">
        <div class="space-y-1.5">
          <label for="email" class="text-sm font-medium">Email</label>
          <div class="relative">
            <MailIcon class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              id="email"
              v-model="email"
              v-bind="emailAttrs"
              type="email"
              autocomplete="email"
              autofocus
              placeholder="anda@contoh.com"
              class="pl-8"
              :aria-invalid="!!errors.email"
              :aria-describedby="errors.email ? 'email-error' : undefined"
            />
          </div>
          <p v-if="errors.email" id="email-error" class="text-sm text-destructive">{{ errors.email }}</p>
        </div>
        <div class="space-y-1.5">
          <div class="flex items-center justify-between">
            <label for="password" class="text-sm font-medium">Kata sandi</label>
            <RouterLink to="/forgot-password" class="text-xs text-primary-700 hover:underline">Lupa kata sandi?</RouterLink>
          </div>
          <div class="relative">
            <LockIcon class="pointer-events-none absolute top-1/2 left-2.5 z-10 size-4 -translate-y-1/2 text-muted-foreground" />
            <PasswordInput
              id="password"
              v-model="password"
              v-bind="passwordAttrs"
              autocomplete="current-password"
              class="pl-8"
              :aria-invalid="!!errors.password"
              :aria-describedby="errors.password ? 'password-error' : undefined"
            />
          </div>
          <p v-if="errors.password" id="password-error" class="text-sm text-destructive">{{ errors.password }}</p>
        </div>

        <Button type="submit" class="w-full" :disabled="isSubmitting">
          <Loader2Icon v-if="isSubmitting" class="size-4 animate-spin" />
          {{ isSubmitting ? 'Sedang masuk…' : 'Masuk' }}
        </Button>
      </form>

      <p class="mt-6 text-center text-sm text-muted-foreground">
        Siswa baru?
        <RouterLink to="/register" class="font-medium text-primary-700 hover:underline">Daftar dengan kode undangan</RouterLink>
      </p>
    </CardContent>
  </Card>
</template>
