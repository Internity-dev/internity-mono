<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { toast } from 'vue-sonner'
import { ArrowLeftIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import PasswordInput from '@/components/shared/PasswordInput.vue'

const schema = toTypedSchema(
  z
    .object({
      current_password: z.string().min(1, 'Current password is required'),
      new_password: z.string().min(8, 'At least 8 characters').max(72, 'At most 72 characters'),
      new_password_confirmation: z.string().min(1, 'Please confirm your new password'),
    })
    .refine((v) => v.new_password === v.new_password_confirmation, {
      message: 'Passwords do not match',
      path: ['new_password_confirmation'],
    }),
)

const { defineField, handleSubmit, errors, setFieldError } = useForm({ validationSchema: schema })
const [currentPassword, currentPasswordAttrs] = defineField('current_password')
const [newPassword, newPasswordAttrs] = defineField('new_password')
const [newPasswordConfirmation, newPasswordConfirmationAttrs] = defineField('new_password_confirmation')

const isSubmitting = ref(false)
const auth = useAuthStore()
const router = useRouter()

// Changing your password revokes every session server-side (see
// identity.Service.ChangePassword) — including this one — so we clear local
// state and send the user back to /login rather than pretending they're
// still authenticated.
const onSubmit = handleSubmit(async (values) => {
  isSubmitting.value = true
  try {
    await http.put('/change-password', {
      current_password: values.current_password,
      new_password: values.new_password,
      new_password_confirmation: values.new_password_confirmation,
    })
    toast.success('Password changed. Log in again')
    auth.clear()
    router.push('/login')
  } catch (err: unknown) {
    const response = (err as { response?: { data?: { message?: string; error?: { details?: { field?: string; issue: string }[] } } } })?.response
    const field = response?.data?.error?.details?.[0]?.field
    if (field === 'current_password') {
      setFieldError('current_password', response?.data?.message ?? 'Current password is incorrect')
    } else {
      toast.error(response?.data?.message ?? 'Could not change password')
    }
  } finally {
    isSubmitting.value = false
  }
})
</script>

<template>
  <div class="mx-auto max-w-md space-y-6">
    <Button as-child variant="ghost" size="sm" class="-ml-2">
      <RouterLink :to="{ name: 'profile' }">
        <ArrowLeftIcon class="size-4" />
        Back to profile
      </RouterLink>
    </Button>

    <Card>
      <CardHeader>
        <CardTitle as="h1">Change password</CardTitle>
        <CardDescription>Choose a new password.</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <label for="current_password" class="text-sm font-medium">Current password</label>
            <PasswordInput
              id="current_password"
              v-model="currentPassword"
              v-bind="currentPasswordAttrs"
              autocomplete="current-password"
            />
            <p v-if="errors.current_password" class="text-sm text-destructive">{{ errors.current_password }}</p>
          </div>

          <div class="space-y-1.5">
            <label for="new_password" class="text-sm font-medium">New password</label>
            <PasswordInput
              id="new_password"
              v-model="newPassword"
              v-bind="newPasswordAttrs"
              autocomplete="new-password"
            />
            <p v-if="errors.new_password" class="text-sm text-destructive">{{ errors.new_password }}</p>
          </div>

          <div class="space-y-1.5">
            <label for="new_password_confirmation" class="text-sm font-medium">Confirm new password</label>
            <PasswordInput
              id="new_password_confirmation"
              v-model="newPasswordConfirmation"
              v-bind="newPasswordConfirmationAttrs"
              autocomplete="new-password"
            />
            <p v-if="errors.new_password_confirmation" class="text-sm text-destructive">
              {{ errors.new_password_confirmation }}
            </p>
          </div>

          <Button type="submit" class="w-full" :disabled="isSubmitting">
            {{ isSubmitting ? 'Updating…' : 'Update password' }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>
