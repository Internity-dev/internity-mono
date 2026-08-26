<script setup lang="ts">
import { ref } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { useRouter, RouterLink } from 'vue-router'
import { toast } from 'vue-sonner'
import { Loader2Icon } from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'
import { errorMessage, fieldErrors, retryAfterSeconds } from '@/lib/api-error'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import PasswordInput from '@/components/shared/PasswordInput.vue'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'

const schema = toTypedSchema(
  z
    .object({
      name: z.string().min(2, 'Masukkan nama lengkap Anda').max(255),
      email: z.string().min(1, 'Email wajib diisi').email('Masukkan email yang valid'),
      password: z.string().min(8, 'Minimal 8 karakter').max(72),
      password_confirmation: z.string().min(1, 'Konfirmasi kata sandi wajib diisi'),
      invite_code: z.string().min(1, 'Kode undangan wajib diisi'),
    })
    .refine((v) => v.password === v.password_confirmation, {
      message: 'Kata sandi tidak cocok',
      path: ['password_confirmation'],
    }),
)

const { defineField, handleSubmit, errors, setErrors } = useForm({ validationSchema: schema })
const [name, nameAttrs] = defineField('name')
const [email, emailAttrs] = defineField('email')
const [password, passwordAttrs] = defineField('password')
const [passwordConfirmation, passwordConfirmationAttrs] = defineField('password_confirmation')
const [inviteCode, inviteCodeAttrs] = defineField('invite_code')

const auth = useAuthStore()
const router = useRouter()
const isSubmitting = ref(false)
const serverError = ref<string | null>(null)

const onSubmit = handleSubmit(async (values) => {
  isSubmitting.value = true
  serverError.value = null
  try {
    await auth.register(values)
    toast.success('Akun berhasil dibuat')
    router.push('/dashboard')
  } catch (err: unknown) {
    const fields = fieldErrors(err)
    if (Object.keys(fields).length) setErrors(fields)
    const retryAfter = retryAfterSeconds(err)
    serverError.value = retryAfter
      ? `Terlalu banyak percobaan. Coba lagi dalam sekitar ${retryAfter} detik.`
      : errorMessage(err, 'Pendaftaran gagal.')
  } finally {
    isSubmitting.value = false
  }
})
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle as="h1">Buat akun Anda</CardTitle>
      <CardDescription>Daftar dengan kode undangan dari koordinator sekolah Anda.</CardDescription>
    </CardHeader>
    <CardContent>
      <Alert v-if="serverError" variant="destructive" class="mb-4">
        <AlertTitle>Gagal mendaftar</AlertTitle>
        <AlertDescription>{{ serverError }}</AlertDescription>
      </Alert>

      <form class="space-y-4" novalidate @submit="onSubmit">
        <div class="space-y-1.5">
          <label for="name" class="text-sm font-medium">Nama lengkap</label>
          <Input
            id="name"
            v-model="name"
            v-bind="nameAttrs"
            autocomplete="name"
            autofocus
            :aria-invalid="!!errors.name"
            :aria-describedby="errors.name ? 'name-error' : undefined"
          />
          <p v-if="errors.name" id="name-error" class="text-sm text-destructive">{{ errors.name }}</p>
        </div>
        <div class="space-y-1.5">
          <label for="email" class="text-sm font-medium">Email</label>
          <Input
            id="email"
            v-model="email"
            v-bind="emailAttrs"
            type="email"
            autocomplete="email"
            :aria-invalid="!!errors.email"
            :aria-describedby="errors.email ? 'email-error' : undefined"
          />
          <p v-if="errors.email" id="email-error" class="text-sm text-destructive">{{ errors.email }}</p>
        </div>
        <div class="space-y-1.5">
          <label for="password" class="text-sm font-medium">Kata sandi</label>
          <PasswordInput
            id="password"
            v-model="password"
            v-bind="passwordAttrs"
            autocomplete="new-password"
            :aria-invalid="!!errors.password"
            :aria-describedby="errors.password ? 'password-error' : undefined"
          />
          <p v-if="errors.password" id="password-error" class="text-sm text-destructive">{{ errors.password }}</p>
        </div>
        <div class="space-y-1.5">
          <label for="password_confirmation" class="text-sm font-medium">Konfirmasi kata sandi</label>
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
        <div class="space-y-1.5">
          <label for="invite_code" class="text-sm font-medium">Kode undangan</label>
          <Input
            id="invite_code"
            v-model="inviteCode"
            v-bind="inviteCodeAttrs"
            :aria-invalid="!!errors.invite_code"
            :aria-describedby="errors.invite_code ? 'invite-code-error' : undefined"
          />
          <p v-if="errors.invite_code" id="invite-code-error" class="text-sm text-destructive">{{ errors.invite_code }}</p>
        </div>

        <Button type="submit" class="w-full" :disabled="isSubmitting">
          <Loader2Icon v-if="isSubmitting" class="size-4 animate-spin" />
          {{ isSubmitting ? 'Membuat akun…' : 'Buat akun' }}
        </Button>
      </form>

      <p class="mt-4 text-center text-sm text-muted-foreground">
        Sudah punya akun?
        <RouterLink to="/login" class="font-medium text-primary-700 hover:underline">Masuk</RouterLink>
      </p>
    </CardContent>
  </Card>
</template>
