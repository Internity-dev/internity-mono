<script setup lang="ts">
import { ref } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { RouterLink } from 'vue-router'
import { Loader2Icon } from '@lucide/vue'
import { http } from '@/lib/http'
import { errorMessage, retryAfterSeconds } from '@/lib/api-error'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'

const schema = toTypedSchema(z.object({ email: z.string().min(1, 'Email wajib diisi').email('Masukkan email yang valid') }))
const { defineField, handleSubmit, errors } = useForm({ validationSchema: schema })
const [email, emailAttrs] = defineField('email')

const isSubmitting = ref(false)
const submitted = ref(false)
const serverError = ref<string | null>(null)

const onSubmit = handleSubmit(async (values) => {
  isSubmitting.value = true
  serverError.value = null
  try {
    await http.post('/auth/forgot-password', values)
    submitted.value = true
  } catch (err: unknown) {
    const retryAfter = retryAfterSeconds(err)
    serverError.value = retryAfter
      ? `Terlalu banyak percobaan. Coba lagi dalam sekitar ${retryAfter} detik.`
      : errorMessage(err, 'Gagal mengirim tautan reset. Coba lagi.')
  } finally {
    isSubmitting.value = false
  }
})
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle as="h1">Lupa kata sandi</CardTitle>
      <CardDescription>Kami akan mengirimkan tautan untuk mengatur ulang kata sandi Anda.</CardDescription>
    </CardHeader>
    <CardContent>
      <div v-if="submitted" class="space-y-4 text-sm">
        <p>Jika email tersebut terdaftar, tautan reset telah dikirim. Periksa kotak masuk Anda.</p>
        <RouterLink to="/login" class="font-medium text-primary-700 hover:underline">Kembali ke halaman masuk</RouterLink>
      </div>
      <template v-else>
        <Alert v-if="serverError" variant="destructive" class="mb-4">
          <AlertTitle>Gagal mengirim</AlertTitle>
          <AlertDescription>{{ serverError }}</AlertDescription>
        </Alert>

        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <label for="email" class="text-sm font-medium">Email</label>
            <Input
              id="email"
              v-model="email"
              v-bind="emailAttrs"
              type="email"
              autocomplete="email"
              autofocus
              :aria-invalid="!!errors.email"
              :aria-describedby="errors.email ? 'email-error' : undefined"
            />
            <p v-if="errors.email" id="email-error" class="text-sm text-destructive">{{ errors.email }}</p>
          </div>
          <Button type="submit" class="w-full" :disabled="isSubmitting">
            <Loader2Icon v-if="isSubmitting" class="size-4 animate-spin" />
            {{ isSubmitting ? 'Mengirim…' : 'Kirim tautan reset' }}
          </Button>
          <p class="text-center text-sm text-muted-foreground">
            <RouterLink to="/login" class="font-medium text-primary-700 hover:underline">Kembali ke halaman masuk</RouterLink>
          </p>
        </form>
      </template>
    </CardContent>
  </Card>
</template>
