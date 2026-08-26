<script setup lang="ts">
import { computed, ref } from 'vue'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import axios from 'axios'
import { toast } from 'vue-sonner'
import { PlusIcon, PencilIcon, Trash2Icon } from '@lucide/vue'
import { http } from '@/lib/http'
import type { ApiSuccess } from '@/types/api'
import type { CreateFaqPayload, Faq, FaqPatch } from '@/types/content'
import { normalizeKeys } from '@/types/content'
import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'

const queryClient = useQueryClient()

const listQuery = useQuery({
  queryKey: ['faqs'],
  queryFn: async () => {
    // FAQ (apps/api/.../content/domain.go) carries no `json` tags, so the
    // raw response is PascalCase — normalizeKeys() converts it to the
    // snake_case shape this view consumes.
    const res = await http.get<ApiSuccess<unknown[]>>('/faqs')
    return normalizeKeys<Faq[]>(res.data.data)
  },
})

const items = computed(() => [...(listQuery.data.value ?? [])].sort((a, b) => a.sort_order - b.sort_order))

const columns: Column[] = [
  { key: 'sort_order', label: 'Order' },
  { key: 'question', label: 'Question' },
  { key: 'answer', label: 'Answer' },
  { key: 'actions', label: '' },
]

function truncate(value: string, len = 60) {
  return value.length > len ? `${value.slice(0, len)}…` : value
}

// --- create / edit dialog ---
const dialogOpen = ref(false)
const editing = ref<Faq | null>(null)

const formSchema = toTypedSchema(
  z.object({
    question: z.string().min(1, 'Question is required'),
    answer: z.string().min(1, 'Answer is required'),
    sort_order: z.coerce.number().int().min(0).optional(),
  }),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: formSchema,
  initialValues: { question: '', answer: '', sort_order: 0 },
})
const [question, questionAttrs] = defineField('question')
const [answer, answerAttrs] = defineField('answer')
const [sortOrder, sortOrderAttrs] = defineField('sort_order')

function openCreate() {
  editing.value = null
  resetForm({ values: { question: '', answer: '', sort_order: items.value.length } })
  dialogOpen.value = true
}

function openEdit(row: Faq) {
  editing.value = row
  resetForm({ values: { question: row.question, answer: row.answer, sort_order: row.sort_order } })
  dialogOpen.value = true
}

function handle422(err: unknown) {
  if (axios.isAxiosError(err) && err.response?.status === 422) {
    toast.error(err.response.data?.message ?? 'Please check the form for errors')
  }
}

const createMutation = useMutation({
  mutationFn: (payload: CreateFaqPayload) => http.post('/faqs', payload),
  onSuccess: () => {
    toast.success('FAQ created')
    queryClient.invalidateQueries({ queryKey: ['faqs'] })
    dialogOpen.value = false
  },
  onError: handle422,
})

const updateMutation = useMutation({
  mutationFn: ({ id, patch }: { id: number; patch: FaqPatch }) => http.put(`/faqs/${id}`, patch),
  onSuccess: () => {
    toast.success('FAQ updated')
    queryClient.invalidateQueries({ queryKey: ['faqs'] })
    dialogOpen.value = false
  },
  onError: handle422,
})

const onSubmit = handleSubmit((values) => {
  if (editing.value) {
    updateMutation.mutate({
      id: editing.value.id,
      patch: { question: values.question, answer: values.answer, sort_order: values.sort_order },
    })
    return
  }
  createMutation.mutate({ question: values.question, answer: values.answer, sort_order: values.sort_order })
})

const isSaving = computed(() => createMutation.isPending.value || updateMutation.isPending.value)

// --- delete ---
const deleteTarget = ref<Faq | null>(null)
const deleteMutation = useMutation({
  mutationFn: (id: number) => http.delete(`/faqs/${id}`),
  onSuccess: () => {
    toast.success('FAQ deleted')
    queryClient.invalidateQueries({ queryKey: ['faqs'] })
    deleteTarget.value = null
  },
})
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Manage FAQ" description="Frequently asked questions shown to everyone, logged in or not.">
      <template #actions>
        <Button @click="openCreate">
          <PlusIcon class="size-4" />
          New FAQ
        </Button>
      </template>
    </PageHeader>

    <DataTable
      :columns="columns"
      :rows="items"
      :is-loading="listQuery.isLoading.value"
      empty-title="No FAQs yet"
      empty-description="Add the questions applicants and students ask most often."
    >
      <template #cell-sort_order="{ row }">
        <span class="tabular-nums text-muted-foreground">{{ row.sort_order }}</span>
      </template>
      <template #cell-answer="{ row }">
        <span class="text-muted-foreground">{{ truncate(row.answer) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="flex justify-end gap-1">
          <Button variant="ghost" size="icon-sm" aria-label="Edit FAQ" @click="openEdit(row)">
            <PencilIcon class="size-4" />
          </Button>
          <Button variant="ghost" size="icon-sm" aria-label="Delete FAQ" @click="deleteTarget = row">
            <Trash2Icon class="size-4 text-destructive" />
          </Button>
        </div>
      </template>
    </DataTable>

    <Dialog v-model:open="dialogOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ editing ? 'Edit FAQ' : 'New FAQ' }}</DialogTitle>
          <DialogDescription>{{ editing ? 'Update this entry.' : 'Visible to everyone, including logged-out visitors.' }}</DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <Label for="faq-question">Question</Label>
            <Textarea id="faq-question" v-model="question" v-bind="questionAttrs" rows="2" />
            <p v-if="errors.question" class="text-sm text-destructive">{{ errors.question }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="faq-answer">Answer</Label>
            <Textarea id="faq-answer" v-model="answer" v-bind="answerAttrs" rows="4" />
            <p v-if="errors.answer" class="text-sm text-destructive">{{ errors.answer }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="faq-sort">Sort order</Label>
            <Input id="faq-sort" v-model="sortOrder" v-bind="sortOrderAttrs" type="number" min="0" />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" :disabled="isSaving" @click="dialogOpen = false">Cancel</Button>
            <Button type="submit" :disabled="isSaving">{{ isSaving ? 'Saving…' : 'Save' }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <ConfirmDialog
      :open="!!deleteTarget"
      title="Delete FAQ?"
      :description="`This removes '${deleteTarget?.question}'.`"
      confirm-label="Delete"
      :is-loading="deleteMutation.isPending.value"
      @update:open="(v) => !v && (deleteTarget = null)"
      @confirm="deleteTarget && deleteMutation.mutate(deleteTarget.id)"
    />
  </div>
</template>
