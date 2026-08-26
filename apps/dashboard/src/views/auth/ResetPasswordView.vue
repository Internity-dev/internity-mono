<script setup lang="ts">
import { ref } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { toast } from 'vue-sonner'
import { Loader2Icon } from '@lucide/vue'
import { http } from '@/lib/http'
import { errorMessage, retryAfterSeconds } from '@/lib/api-error'
import { Button } from '@/components/ui/button'
import PasswordInput from '@/components/shared/PasswordInput.vue'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'

const route = useRoute()
const router = useRouter()
const token = typeof route.query.token === 'string' ? route.query.token : ''

const schema = toTypedSchema(
  z
    .object({
      password: z.string().min(8, 'Minimal 8 karakter').max(72),
      password_confirmation: z.string().min(1, 'Konfirmasi kata sandi wajib diisi'),
    })
    .refine((v) => v.password === v.password_confirmation, {
      message: 'Kata sandi tidak cocok',
      path: ['password_confirmation'],
    }),
)
const { defineField, handleSubmit, errors } = useForm({ validationSchema: schema })
const [password, passwordAttrs] = defineField('password')
const [passwordConfirmation, passwordConfirmationAttrs] = defineField('password_confirmation')

const isSubmitting = ref(false)
const serverError = ref<string | null>(null)

const onSubmit = handleSubmit(async (values) => {
  isSubmitting.value = true
  serverError.value = null
  try {
    await http.post('/auth/reset-password', { token, ...values })
    toast.success('Kata sandi berhasil diatur ulang. Silakan masuk kembali')
    router.push('/login')
  } catch (err: unknown) {
    const retryAfter = retryAfterSeconds(err)
    serverError.value = retryAfter
      ? `Terlalu banyak percobaan. Coba lagi dalam sekitar ${retryAfter} detik.`
      : errorMessage(err, 'Gagal mengatur ulang kata sandi.')
  } finally {
    isSubmitting.value = false
  }
})
</script>

<template>
  <Card>
    <template v-if="!token">
      <CardHeader>
        <CardTitle as="h1">Tautan tidak valid</CardTitle>
        <CardDescription>Tautan reset kata sandi ini tidak lengkap atau sudah kedaluwarsa.</CardDescription>
      </CardHeader>
      <CardContent>
        <RouterLink to="/forgot-password" class="font-medium text-primary-700 hover:underline">
          Minta tautan reset baru
        </RouterLink>
      </CardContent>
    </template>
    <template v-else>
      <CardHeader>
        <CardTitle as="h1">Atur ulang kata sandi</CardTitle>
        <CardDescription>Pilih kata sandi baru untuk akun Anda.</CardDescription>
      </CardHeader>
      <CardContent>
        <Alert v-if="serverError" variant="destructive" class="mb-4">
          <AlertTitle>Gagal mengatur ulang</AlertTitle>
          <AlertDescription>{{ serverError }}</AlertDescription>
        </Alert>

        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <label for="password" class="text-sm font-medium">Kata sandi baru</label>
            <PasswordInput
              id="password"
              v-model="password"
              v-bind="passwordAttrs"
              autocomplete="new-password"
              autofocus
              :aria-invalid="!!errors.password"
              :aria-describedby="errors.password ? 'password-error' : undefined"
            />
            <p v-if="errors.password" id="password-error" class="text-sm text-destructive">{{ errors.password }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="password_confirmation" class="text-sm font-medium">Konfirmasi kata sandi baru</label>
            <PasswordInput
              id="password_confirmation"
              v-model="passwordConfirmation"
              v-bind="passwordConfirmationAttrs"
              autocomplete="new-password"
              :aria-invalid="!!errors.password_confirmation"
              :aria-describedby="errors.password_confirmation ? 'password-confirmation-error' : undefined"
            />
            <p v-if="errors.password_confirmation" id="password-confirmation-error" class="text-sm text-destructive">
              {{ errors.password_confirmation }}
            </p>
          </div>

          <Button type="submit" class="w-full" :disabled="isSubmitting">
            <Loader2Icon v-if="isSubmitting" class="size-4 animate-spin" />
            {{ isSubmitting ? 'Mengatur ulang…' : 'Atur ulang kata sandi' }}
          </Button>
          <p class="text-center text-sm text-muted-foreground">
            <RouterLink to="/login" class="font-medium text-primary-700 hover:underline">Kembali ke halaman masuk</RouterLink>
          </p>
        </form>
      </CardContent>
    </template>
  </Card>
</template>
