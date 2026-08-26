<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import axios from 'axios'
import { toast } from 'vue-sonner'
import { DownloadIcon, FileSpreadsheetIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import type { ApiSuccess } from '@/types/api'
import PageHeader from '@/components/shared/PageHeader.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

// orgs.Department / orgs.Company (apps/api/.../orgs/domain.go) carry no
// `json` tags, so these pickers normalize the raw PascalCase (ID, Name)
// response.
interface Department {
  id: number
  name: string
}
interface Company {
  id: number
  name: string
}
function pascalToSnake(key: string): string {
  return key.replace(/([a-z0-9])([A-Z])/g, '$1_$2').toLowerCase()
}
function normalizeKeys<T>(raw: unknown): T {
  if (Array.isArray(raw)) return raw.map((item) => normalizeKeys(item)) as unknown as T
  if (raw !== null && typeof raw === 'object') {
    const out: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(raw as Record<string, unknown>)) out[pascalToSnake(key)] = value
    return out as T
  }
  return raw as T
}

/**
 * Both export endpoints stream an .xlsx binary rather than the usual JSON
 * envelope, so this app requests them with `responseType: 'blob'`. When the
 * request fails, axios still hands back the response body as a Blob (not
 * parsed JSON) — the global http.ts interceptor can't read `.message` off a
 * Blob, so 400/404/409 errors would otherwise fail silently. This reads the
 * blob as text and parses it as the API's error envelope so a toast can
 * still show something useful.
 */
async function extractBlobErrorMessage(err: unknown): Promise<string> {
  if (axios.isAxiosError(err) && err.response?.data instanceof Blob) {
    try {
      const text = await err.response.data.text()
      const parsed = JSON.parse(text) as { message?: string }
      return parsed.message ?? 'Export failed'
    } catch {
      return 'Export failed'
    }
  }
  if (axios.isAxiosError(err)) return (err.response?.data as { message?: string } | undefined)?.message ?? 'Export failed'
  return 'Export failed'
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

const auth = useAuthStore()
const isAdmin = computed(() => auth.user?.role === 'admin')

// A coordinator's school picker has zero effect: it only feeds the
// /departments picker call below, and that endpoint pins a non-admin to
// their own school regardless of what school_id is requested (see
// orgs/service.go's scopedSchoolFilter). So, like DepartmentsView/
// CompaniesView, it's hidden for them entirely and schoolId is pinned to
// their own school.
const schoolIdInput = ref<string>(auth.user?.school_id ? String(auth.user.school_id) : '')
const schoolId = computed(() => {
  if (!isAdmin.value) return auth.user?.school_id
  const n = Number(schoolIdInput.value)
  return schoolIdInput.value !== '' && Number.isFinite(n) && n > 0 ? n : undefined
})

const departmentsQuery = useQuery({
  queryKey: computed(() => ['departments-picker', schoolId.value]),
  queryFn: async () => {
    const res = await http.get<ApiSuccess<unknown[]>>('/departments', { params: { school_id: schoolId.value, limit: 100 } })
    return normalizeKeys<Department[]>(res.data.data)
  },
  enabled: computed(() => schoolId.value !== undefined),
})
const departments = computed(() => departmentsQuery.data.value ?? [])

// --- Student roster export ---
const rosterDepartmentId = ref<number | undefined>(undefined)
const isRosterDownloading = ref(false)

async function downloadRoster() {
  if (!rosterDepartmentId.value) {
    toast.error('Pick a department first')
    return
  }
  isRosterDownloading.value = true
  try {
    const res = await http.get('/exports/students', {
      params: { department_id: rosterDepartmentId.value },
      responseType: 'blob',
    })
    downloadBlob(res.data, 'students.xlsx')
    toast.success('Student roster downloaded')
  } catch (err) {
    toast.error(await extractBlobErrorMessage(err))
  } finally {
    isRosterDownloading.value = false
  }
}

// --- Presence export ---
const presenceDepartmentId = ref<number | undefined>(undefined)
const presenceCompanyId = ref<number | undefined>(undefined)
const presenceUserId = ref('')
const isPresenceDownloading = ref(false)

const presenceCompaniesQuery = useQuery({
  queryKey: computed(() => ['companies-picker', presenceDepartmentId.value]),
  queryFn: async () => {
    const res = await http.get<ApiSuccess<unknown[]>>('/companies', {
      params: { department_id: presenceDepartmentId.value, limit: 100 },
    })
    return normalizeKeys<Company[]>(res.data.data)
  },
  enabled: computed(() => presenceDepartmentId.value !== undefined),
})
const presenceCompanies = computed(() => presenceCompaniesQuery.data.value ?? [])

watch(presenceDepartmentId, () => {
  presenceCompanyId.value = undefined
})

async function downloadPresence() {
  if (!presenceCompanyId.value) {
    toast.error('Pick a company first')
    return
  }
  if (!presenceUserId.value) {
    toast.error('Enter a student user ID')
    return
  }
  isPresenceDownloading.value = true
  try {
    const res = await http.get('/exports/presence', {
      params: { user_id: presenceUserId.value, company_id: presenceCompanyId.value },
      responseType: 'blob',
    })
    downloadBlob(res.data, `presence-${presenceCompanyId.value}.xlsx`)
    toast.success('Presence report downloaded')
  } catch (err) {
    toast.error(await extractBlobErrorMessage(err))
  } finally {
    isPresenceDownloading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Reports" description="Export data to Excel for offline use or record-keeping." />

    <Card v-if="isAdmin">
      <CardContent class="flex flex-wrap items-end gap-3">
        <div class="space-y-1.5">
          <Label for="school-id">School ID</Label>
          <Input id="school-id" v-model="schoolIdInput" type="number" placeholder="e.g. 1" class="w-40" />
        </div>
      </CardContent>
    </Card>

    <div class="grid gap-4 md:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle class="flex items-center gap-2">
            <FileSpreadsheetIcon class="size-4" />
            Student roster
          </CardTitle>
          <CardDescription>Export every student in a department to an Excel file.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="space-y-1.5">
            <Label for="roster-department">Department</Label>
            <Select v-model="rosterDepartmentId">
              <SelectTrigger id="roster-department" class="w-full">
                <SelectValue placeholder="Select department" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button :disabled="!rosterDepartmentId || isRosterDownloading" @click="downloadRoster">
            <DownloadIcon class="size-4" />
            {{ isRosterDownloading ? 'Preparing file…' : 'Download roster' }}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="flex items-center gap-2">
            <FileSpreadsheetIcon class="size-4" />
            Presence report
          </CardTitle>
          <CardDescription>Export one student's attendance at a company to an Excel file.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div class="space-y-1.5">
              <Label for="presence-department">Department</Label>
              <Select v-model="presenceDepartmentId">
                <SelectTrigger id="presence-department" class="w-full">
                  <SelectValue placeholder="Select department" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-1.5">
              <Label for="presence-company">Company</Label>
              <Select v-model="presenceCompanyId" :disabled="!presenceDepartmentId">
                <SelectTrigger id="presence-company" class="w-full">
                  <SelectValue placeholder="Select company" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="c in presenceCompanies" :key="c.id" :value="c.id">{{ c.name }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <div class="space-y-1.5">
            <Label for="presence-user">Student user ID</Label>
            <Input id="presence-user" v-model="presenceUserId" placeholder="00000000-0000-0000-0000-000000000000" />
          </div>
          <Button :disabled="!presenceCompanyId || !presenceUserId || isPresenceDownloading" @click="downloadPresence">
            <DownloadIcon class="size-4" />
            {{ isPresenceDownloading ? 'Preparing file…' : 'Download presence report' }}
          </Button>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
