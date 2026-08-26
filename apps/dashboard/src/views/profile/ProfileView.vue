<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { toast } from 'vue-sonner'
import { KeyIcon, CameraIcon, Loader2Icon, PhoneIcon, CalendarIcon, MapPinIcon, FileTextIcon, TagIcon, UserRoundIcon } from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'
import { http } from '@/lib/http'
import { avatarUrl } from '@/lib/avatar'
import type { ApiSuccess, User } from '@/types/api'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const auth = useAuthStore()
const user = computed(() => auth.user)

const activeTab = ref<'overview' | 'edit'>('overview')

const initials = computed(() => {
  const name = user.value?.name ?? ''
  const parts = name.split(/\s+/).filter(Boolean).slice(0, 2)
  return parts.map((p) => p[0]?.toUpperCase()).join('') || '?'
})

const avatarSrc = computed(() => avatarUrl(user.value?.avatar_key))

const skillsList = computed(() =>
  (user.value?.skills ?? '')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean),
)

function formatDate(value: string) {
  return new Date(value).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' })
}

// --- Avatar upload ---
const fileInput = ref<HTMLInputElement | null>(null)
const uploadingAvatar = ref(false)

async function onAvatarSelected(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  uploadingAvatar.value = true
  try {
    const form = new FormData()
    form.append('avatar', file)
    const res = await http.post<ApiSuccess<{ user: User }>>('/avatars', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    auth.user = res.data.data.user
    toast.success('Photo updated')
  } catch (err: unknown) {
    const message = (err as { response?: { data?: { message?: string } } })?.response?.data?.message ?? 'Could not upload photo'
    toast.error(message)
  } finally {
    uploadingAvatar.value = false
    if (fileInput.value) fileInput.value.value = ''
  }
}

const profileSchema = toTypedSchema(
  z.object({
    name: z.string().min(2, 'Enter your full name').max(255),
    phone: z.string().max(30, 'Too long').optional().or(z.literal('')),
    address: z.string().max(500, 'Too long').optional().or(z.literal('')),
    bio: z.string().max(1000, 'Too long').optional().or(z.literal('')),
    skills: z.string().max(500, 'Too long').optional().or(z.literal('')),
    gender: z.union([z.literal('male'), z.literal('female'), z.literal('')]).optional(),
    date_of_birth: z.string().optional().or(z.literal('')),
  }),
)

const { defineField, handleSubmit, errors } = useForm({
  validationSchema: profileSchema,
  initialValues: {
    name: user.value?.name ?? '',
    phone: user.value?.phone ?? '',
    address: user.value?.address ?? '',
    bio: user.value?.bio ?? '',
    skills: user.value?.skills ?? '',
    gender: user.value?.gender ?? '',
    date_of_birth: user.value?.date_of_birth ? user.value.date_of_birth.slice(0, 10) : '',
  },
})
const [name, nameAttrs] = defineField('name')
const [phone, phoneAttrs] = defineField('phone')
const [address, addressAttrs] = defineField('address')
const [bio, bioAttrs] = defineField('bio')
const [skills, skillsAttrs] = defineField('skills')
const [gender, genderAttrs] = defineField('gender')
const [dateOfBirth, dateOfBirthAttrs] = defineField('date_of_birth')

const isSubmitting = ref(false)

const onSubmit = handleSubmit(async (values) => {
  isSubmitting.value = true
  try {
    const res = await http.put<ApiSuccess<{ user: User }>>('/change-profile', {
      name: values.name,
      phone: values.phone || null,
      address: values.address || null,
      bio: values.bio || null,
      skills: values.skills || null,
      gender: values.gender || null,
      date_of_birth: values.date_of_birth || null,
    })
    // Update the store directly so the navbar/avatar reflect the change
    // immediately, without a full page reload.
    auth.user = res.data.data.user
    toast.success('Profile updated')
  } catch (err: unknown) {
    const message = (err as { response?: { data?: { message?: string } } })?.response?.data?.message ?? 'Could not update profile'
    toast.error(message)
  } finally {
    isSubmitting.value = false
  }
})
</script>

<template>
  <div v-if="!user" class="space-y-4">
    <Skeleton class="h-56 w-full" />
  </div>

  <div v-else class="space-y-6">
    <div class="space-y-1">
      <h1 class="font-display text-2xl font-semibold tracking-tight text-foreground">My Profile</h1>
      <p class="text-sm text-muted-foreground">View and update your account details.</p>
    </div>

    <div class="flex gap-2">
      <Button :variant="activeTab === 'overview' ? 'default' : 'outline'" size="sm" @click="activeTab = 'overview'">Overview</Button>
      <Button :variant="activeTab === 'edit' ? 'default' : 'outline'" size="sm" @click="activeTab = 'edit'">Edit Profile</Button>
    </div>

    <template v-if="activeTab === 'overview'">
      <div class="space-y-4">
        <Card class="overflow-hidden p-0">
          <div class="relative">
            <div class="h-20 bg-linear-to-br from-primary-500 to-primary-800 sm:h-24" />
            <div class="absolute bottom-0 left-4 translate-y-1/2">
              <div class="group relative">
                <Avatar class="size-20 ring-4 ring-card sm:size-24">
                  <AvatarImage v-if="avatarSrc" :src="avatarSrc" :alt="user.name" />
                  <AvatarFallback class="text-2xl">{{ initials }}</AvatarFallback>
                </Avatar>
                <button
                  type="button"
                  class="absolute right-0 bottom-0 flex size-7 items-center justify-center rounded-full bg-primary text-primary-foreground ring-2 ring-card transition-transform hover:scale-105 disabled:opacity-60 sm:size-8"
                  :disabled="uploadingAvatar"
                  aria-label="Change photo"
                  @click="fileInput?.click()"
                >
                  <Loader2Icon v-if="uploadingAvatar" class="size-4 animate-spin" />
                  <CameraIcon v-else class="size-4" />
                </button>
                <input ref="fileInput" type="file" accept="image/*" class="hidden" @change="onAvatarSelected" />
              </div>
            </div>
          </div>

          <CardContent class="px-4 pt-12 pb-4 sm:pt-14">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h2 class="text-lg font-semibold text-foreground">{{ user.name }}</h2>
                <p class="text-sm text-muted-foreground">{{ user.email }}</p>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <Badge variant="secondary" class="capitalize">{{ user.role }}</Badge>
                <Badge v-if="user.nis" variant="outline">NIS {{ user.nis }}</Badge>
                <Badge v-if="!user.is_active" variant="destructive">Inactive</Badge>
              </div>
            </div>
          </CardContent>

          <Separator />

          <CardContent class="grid gap-5 py-4 sm:grid-cols-2">
            <div class="flex items-start gap-2.5">
              <PhoneIcon class="mt-0.5 size-4 shrink-0 text-muted-foreground" />
              <div>
                <p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Phone</p>
                <p class="text-sm text-foreground">{{ user.phone || '—' }}</p>
              </div>
            </div>
            <div class="flex items-start gap-2.5">
              <UserRoundIcon class="mt-0.5 size-4 shrink-0 text-muted-foreground" />
              <div>
                <p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Gender</p>
                <p class="text-sm text-foreground capitalize">{{ user.gender || '—' }}</p>
              </div>
            </div>
            <div class="flex items-start gap-2.5">
              <CalendarIcon class="mt-0.5 size-4 shrink-0 text-muted-foreground" />
              <div>
                <p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Date of birth</p>
                <p class="text-sm text-foreground">{{ user.date_of_birth ? formatDate(user.date_of_birth) : '—' }}</p>
              </div>
            </div>
            <div class="flex items-start gap-2.5">
              <MapPinIcon class="mt-0.5 size-4 shrink-0 text-muted-foreground" />
              <div>
                <p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Address</p>
                <p class="text-sm text-foreground">{{ user.address || '—' }}</p>
              </div>
            </div>
            <div class="flex items-start gap-2.5 sm:col-span-2">
              <FileTextIcon class="mt-0.5 size-4 shrink-0 text-muted-foreground" />
              <div>
                <p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Bio</p>
                <p class="text-sm whitespace-pre-wrap text-foreground">{{ user.bio || '—' }}</p>
              </div>
            </div>
            <div v-if="skillsList.length" class="flex items-start gap-2.5 sm:col-span-2">
              <TagIcon class="mt-0.5 size-4 shrink-0 text-muted-foreground" />
              <div>
                <p class="mb-1.5 text-xs font-medium tracking-wide text-muted-foreground uppercase">Skills</p>
                <div class="flex flex-wrap gap-1.5">
                  <Badge v-for="s in skillsList" :key="s" variant="outline">{{ s }}</Badge>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Button variant="outline" as-child>
          <RouterLink :to="{ name: 'change-password' }">
            <KeyIcon class="size-4" />
            Change password
          </RouterLink>
        </Button>
      </div>
    </template>

    <template v-else>
      <div>
        <Card>
          <CardContent>
            <form class="space-y-4" novalidate @submit="onSubmit">
              <div class="space-y-1.5">
                <label for="name" class="text-sm font-medium">Full name</label>
                <Input id="name" v-model="name" v-bind="nameAttrs" />
                <p v-if="errors.name" class="text-sm text-destructive">{{ errors.name }}</p>
              </div>

              <div class="grid gap-4 sm:grid-cols-2">
                <div class="space-y-1.5">
                  <label for="phone" class="text-sm font-medium">Phone</label>
                  <Input id="phone" v-model="phone" v-bind="phoneAttrs" autocomplete="tel" />
                  <p v-if="errors.phone" class="text-sm text-destructive">{{ errors.phone }}</p>
                </div>
                <div class="space-y-1.5">
                  <label for="date_of_birth" class="text-sm font-medium">Date of birth</label>
                  <Input id="date_of_birth" v-model="dateOfBirth" v-bind="dateOfBirthAttrs" type="date" />
                  <p v-if="errors.date_of_birth" class="text-sm text-destructive">{{ errors.date_of_birth }}</p>
                </div>
              </div>

              <div class="space-y-1.5">
                <label for="gender" class="text-sm font-medium">Gender</label>
                <Select v-model="gender" v-bind="genderAttrs">
                  <SelectTrigger id="gender" class="w-full"><SelectValue placeholder="Not specified" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="male">Male</SelectItem>
                    <SelectItem value="female">Female</SelectItem>
                  </SelectContent>
                </Select>
                <p v-if="errors.gender" class="text-sm text-destructive">{{ errors.gender }}</p>
              </div>

              <div class="space-y-1.5">
                <label for="address" class="text-sm font-medium">Address</label>
                <Textarea id="address" v-model="address" v-bind="addressAttrs" rows="2" />
                <p v-if="errors.address" class="text-sm text-destructive">{{ errors.address }}</p>
              </div>

              <div class="space-y-1.5">
                <label for="bio" class="text-sm font-medium">Bio</label>
                <Textarea id="bio" v-model="bio" v-bind="bioAttrs" rows="3" />
                <p v-if="errors.bio" class="text-sm text-destructive">{{ errors.bio }}</p>
              </div>

              <div class="space-y-1.5">
                <label for="skills" class="text-sm font-medium">Skills</label>
                <Input id="skills" v-model="skills" v-bind="skillsAttrs" placeholder="Comma-separated, e.g. Figma, HTML, Excel" />
                <p v-if="errors.skills" class="text-sm text-destructive">{{ errors.skills }}</p>
              </div>

              <Button type="submit" :disabled="isSubmitting">
                {{ isSubmitting ? 'Saving…' : 'Save changes' }}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </template>
  </div>
</template>
