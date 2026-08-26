<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import axios from 'axios'
import { toast } from 'vue-sonner'
import { PlusIcon, PencilIcon, Trash2Icon, SendIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import type { ApiSuccess } from '@/types/api'
import type { CreateNewsPayload, News, NewsPatch, NewsScopeType } from '@/types/content'
import { normalizeKeys } from '@/types/content'
import { newsStatus } from '@/lib/status'
import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

// orgs.Department (apps/api/.../orgs/domain.go) carries no `json` tags, so
// this picker normalizes the raw PascalCase (ID, Name) response.
interface Department {
  id: number
  name: string
}
function pascalToSnake(key: string): string {
  return key.replace(/([a-z0-9])([A-Z])/g, '$1_$2').toLowerCase()
}
function normalizePickerKeys<T>(raw: unknown): T {
  if (Array.isArray(raw)) return raw.map((item) => normalizePickerKeys(item)) as unknown as T
  if (raw !== null && typeof raw === 'object') {
    const out: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(raw as Record<string, unknown>)) out[pascalToSnake(key)] = value
    return out as T
  }
  return raw as T
}

const auth = useAuthStore()
const queryClient = useQueryClient()

const schoolId = computed(() => auth.user?.school_id)

const departmentsQuery = useQuery({
  queryKey: computed(() => ['departments-picker', schoolId.value]),
  queryFn: async () => {
    const res = await http.get<ApiSuccess<unknown[]>>('/departments', { params: { school_id: schoolId.value, limit: 100 } })
    return normalizePickerKeys<Department[]>(res.data.data)
  },
  enabled: computed(() => schoolId.value !== undefined),
})
const departments = computed(() => departmentsQuery.data.value ?? [])

// --- list ---
const listQuery = useListQuery<News>(
  'admin-news',
  async (params) => {
    // News (apps/api/.../content/domain.go) carries no `json` tags, so the
    // raw response is PascalCase — normalize just the `data` payload.
    const res = await http.get<ApiSuccess<unknown[]>>('/news/manage', { params })
    return { ...res.data, data: normalizeKeys<News[]>(res.data.data) }
  },
  { defaultSort: 'created_at' },
)

const columns: Column[] = [
  { key: 'title', label: 'Title' },
  { key: 'scope', label: 'Scope' },
  { key: 'status', label: 'Status' },
  { key: 'published_at', label: 'Published', sortable: true },
  { key: 'actions', label: '' },
]

// --- create / edit dialog ---
const dialogOpen = ref(false)
const editing = ref<News | null>(null)

const formSchema = toTypedSchema(
  z
    .object({
      title: z.string().min(2, 'Title is required').max(255),
      content: z.string().min(1, 'Content is required'),
      scope_type: z.enum(['school', 'department']),
      department_id: z.coerce.number().optional(),
    })
    .refine((v) => v.scope_type === 'school' || (v.department_id !== undefined && v.department_id > 0), {
      message: 'Pick a department',
      path: ['department_id'],
    }),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: formSchema,
  initialValues: { title: '', content: '', scope_type: 'school' as NewsScopeType, department_id: undefined },
})
const [title, titleAttrs] = defineField('title')
const [content, contentAttrs] = defineField('content')
const [scopeType, scopeTypeAttrs] = defineField('scope_type')
const [departmentId, departmentIdAttrs] = defineField('department_id')

watch(scopeType, (v) => {
  if (v === 'school') departmentId.value = undefined
})

function openCreate() {
  editing.value = null
  resetForm({ values: { title: '', content: '', scope_type: 'school', department_id: undefined } })
  dialogOpen.value = true
}

function openEdit(row: News) {
  editing.value = row
  resetForm({ values: { title: row.title, content: row.content, scope_type: row.scope_type, department_id: undefined } })
  dialogOpen.value = true
}

function handle422(err: unknown) {
  if (axios.isAxiosError(err) && err.response?.status === 422) {
    toast.error(err.response.data?.message ?? 'Please check the form for errors')
  }
}

const createMutation = useMutation({
  mutationFn: (payload: CreateNewsPayload) => http.post('/news', payload),
  onSuccess: (_res, payload) => {
    toast.success(payload.publish ? 'News published, everyone in scope has been notified' : 'Draft saved')
    queryClient.invalidateQueries({ queryKey: ['admin-news'] })
    dialogOpen.value = false
  },
  onError: handle422,
})

const updateMutation = useMutation({
  mutationFn: ({ id, patch }: { id: number; patch: NewsPatch }) => http.put(`/news/${id}`, patch),
  onSuccess: (_res, { patch }) => {
    toast.success(patch.publish ? 'News published, everyone in scope has been notified' : 'News updated')
    queryClient.invalidateQueries({ queryKey: ['admin-news'] })
    dialogOpen.value = false
  },
  onError: handle422,
})

function submitEdit(values: { title: string; content: string }) {
  if (!editing.value) return
  updateMutation.mutate({ id: editing.value.id, patch: { title: values.title, content: values.content } })
}

function submitCreate(values: { title: string; content: string; scope_type: NewsScopeType; department_id?: number }, publish: boolean) {
  if (!schoolId.value) {
    toast.error('Your account has no school configured')
    return
  }
  createMutation.mutate({
    scope_type: values.scope_type,
    scope_id: values.scope_type === 'school' ? schoolId.value : (values.department_id as number),
    title: values.title,
    content: values.content,
    publish,
  })
}

const onSaveDraft = handleSubmit((v) => submitCreate(v, false))
const onPublish = handleSubmit((v) => submitCreate(v, true))
const onSaveEdit = handleSubmit((v) => submitEdit(v))

const isSaving = computed(() => createMutation.isPending.value || updateMutation.isPending.value)

// --- quick publish for drafts ---
const publishMutation = useMutation({
  mutationFn: (id: number) => http.put(`/news/${id}`, { publish: true }),
  onSuccess: () => {
    toast.success('News published, everyone in scope has been notified')
    queryClient.invalidateQueries({ queryKey: ['admin-news'] })
  },
})

// --- delete ---
const deleteTarget = ref<News | null>(null)
const deleteMutation = useMutation({
  mutationFn: (id: number) => http.delete(`/news/${id}`),
  onSuccess: () => {
    toast.success('News deleted')
    queryClient.invalidateQueries({ queryKey: ['admin-news'] })
    deleteTarget.value = null
  },
})

function formatDate(value: string | null) {
  if (!value) return '—'
  return new Date(value).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Manage News" description="Create, edit, and publish announcements for your school or a department.">
      <template #actions>
        <Button @click="openCreate">
          <PlusIcon class="size-4" />
          New post
        </Button>
      </template>
    </PageHeader>

    <DataTable
      :columns="columns"
      :rows="listQuery.items.value"
      :is-loading="listQuery.isLoading.value"
      :sort="listQuery.sort.value"
      :order="listQuery.order.value"
      empty-title="No news posts yet"
      empty-description="Create your first announcement. Save it as a draft or publish immediately."
      @sort="(key) => listQuery.setParams({ sort: key, order: listQuery.order.value === 'asc' ? 'desc' : 'asc' })"
    >
      <template #cell-scope="{ row }">
        <StatusBadge tone="info" :label="row.scope_type === 'school' ? 'School' : 'Department'" />
      </template>
      <template #cell-status="{ row }">
        <StatusBadge :tone="newsStatus(row.status).tone" :label="newsStatus(row.status).label" />
      </template>
      <template #cell-published_at="{ row }">
        <span class="text-muted-foreground">{{ formatDate(row.published_at) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="flex justify-end gap-1">
          <Button
            v-if="row.status === 'draft'"
            variant="ghost"
            size="icon-sm"
            title="Publish now"
            :disabled="publishMutation.isPending.value"
            @click="publishMutation.mutate(row.id)"
          >
            <SendIcon class="size-4 text-success" />
          </Button>
          <Button variant="ghost" size="icon-sm" aria-label="Edit post" @click="openEdit(row)">
            <PencilIcon class="size-4" />
          </Button>
          <Button variant="ghost" size="icon-sm" aria-label="Delete post" @click="deleteTarget = row">
            <Trash2Icon class="size-4 text-destructive" />
          </Button>
        </div>
      </template>
    </DataTable>

    <ListPagination
      v-if="listQuery.pagination.value"
      :page="listQuery.page.value"
      :limit="listQuery.limit.value"
      :total="listQuery.pagination.value.total"
      @update:page="(p) => listQuery.setParams({ page: p })"
    />

    <Dialog v-model:open="dialogOpen">
      <DialogContent class="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{{ editing ? 'Edit post' : 'New post' }}</DialogTitle>
          <DialogDescription>
            {{ editing ? 'Update the title and content.' : 'Save as a draft, or publish immediately.' }}
          </DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit.prevent>
          <div class="space-y-1.5">
            <Label for="news-title">Title</Label>
            <Input id="news-title" v-model="title" v-bind="titleAttrs" />
            <p v-if="errors.title" class="text-sm text-destructive">{{ errors.title }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="news-content">Content</Label>
            <Textarea id="news-content" v-model="content" v-bind="contentAttrs" rows="6" />
            <p v-if="errors.content" class="text-sm text-destructive">{{ errors.content }}</p>
          </div>

          <template v-if="!editing">
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div class="space-y-1.5">
                <Label for="news-scope">Scope</Label>
                <Select v-model="scopeType" v-bind="scopeTypeAttrs">
                  <SelectTrigger id="news-scope" class="w-full">
                    <SelectValue placeholder="Select scope" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="school">School</SelectItem>
                    <SelectItem value="department">Department</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div class="space-y-1.5">
                <Label for="news-department">Department</Label>
                <Select v-model="departmentId" v-bind="departmentIdAttrs" :disabled="scopeType !== 'department'">
                  <SelectTrigger id="news-department" class="w-full">
                    <SelectValue placeholder="Select department" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</SelectItem>
                  </SelectContent>
                </Select>
                <p v-if="errors.department_id" class="text-sm text-destructive">{{ errors.department_id }}</p>
              </div>
            </div>

            <Alert>
              <AlertDescription>Publishing will notify everyone in the selected scope.</AlertDescription>
            </Alert>

            <DialogFooter>
              <Button type="button" variant="outline" :disabled="isSaving" @click="dialogOpen = false">Cancel</Button>
              <Button type="button" variant="secondary" :disabled="isSaving" @click="onSaveDraft">
                {{ isSaving ? 'Saving…' : 'Save as draft' }}
              </Button>
              <Button type="button" :disabled="isSaving" @click="onPublish">
                {{ isSaving ? 'Publishing…' : 'Publish' }}
              </Button>
            </DialogFooter>
          </template>

          <DialogFooter v-else>
            <Button type="button" variant="outline" :disabled="isSaving" @click="dialogOpen = false">Cancel</Button>
            <Button type="button" :disabled="isSaving" @click="onSaveEdit">
              {{ isSaving ? 'Saving…' : 'Save changes' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <ConfirmDialog
      :open="!!deleteTarget"
      title="Delete this post?"
      :description="`This permanently removes '${deleteTarget?.title}'.`"
      confirm-label="Delete"
      :is-loading="deleteMutation.isPending.value"
      @update:open="(v) => !v && (deleteTarget = null)"
      @confirm="deleteTarget && deleteMutation.mutate(deleteTarget.id)"
    />
  </div>
</template>
