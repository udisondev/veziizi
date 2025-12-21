<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { adminApi } from '@/api/admin'
import type { OrganizationDetail } from '@/types/admin'

// UI Components
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

// Shared Components
import { BackLink, ErrorBanner } from '@/components/shared'

// Icons
import { Check, X, Building2, Users, Mail, Phone, MapPin, Globe, FileText, Calendar } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()

const organization = ref<OrganizationDetail | null>(null)
const isLoading = ref(true)
const error = ref('')
const actionLoading = ref(false)
const showRejectModal = ref(false)
const rejectReason = ref('')

const countryNames: Record<string, string> = {
  RU: 'Россия',
  KZ: 'Казахстан',
  BY: 'Беларусь',
}

const statusNames: Record<string, string> = {
  pending: 'На модерации',
  active: 'Активна',
  suspended: 'Приостановлена',
  rejected: 'Отклонена',
}

const statusVariants: Record<string, 'default' | 'success' | 'warning' | 'destructive' | 'secondary'> = {
  pending: 'warning',
  active: 'success',
  suspended: 'destructive',
  rejected: 'destructive',
}

const roleNames: Record<string, string> = {
  owner: 'Владелец',
  administrator: 'Администратор',
  employee: 'Сотрудник',
}

onMounted(async () => {
  await loadOrganization()
})

async function loadOrganization() {
  isLoading.value = true
  error.value = ''
  try {
    const id = route.params.id as string
    organization.value = await adminApi.getOrganization(id)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function handleApprove() {
  if (!organization.value) return
  actionLoading.value = true
  try {
    await adminApi.approveOrganization(organization.value.id)
    router.push('/admin')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

async function handleReject() {
  if (!organization.value || !rejectReason.value.trim()) return
  actionLoading.value = true
  try {
    await adminApi.rejectOrganization(organization.value.id, { reason: rejectReason.value })
    router.push('/admin')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
    showRejectModal.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-slate-900">
    <!-- Header -->
    <header class="bg-slate-800 border-b border-slate-700">
      <div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <BackLink to="/admin/organizations" label="Назад к списку" class="text-indigo-400 hover:text-indigo-300" />
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Loading -->
      <div v-if="isLoading" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
      </div>

      <!-- Error -->
      <ErrorBanner v-else-if="error" :message="error" @retry="loadOrganization" />

      <!-- Organization Details -->
      <div v-else-if="organization" class="space-y-6">
        <!-- Header -->
        <div class="flex flex-col sm:flex-row sm:justify-between sm:items-start gap-4">
          <div>
            <h1 class="text-2xl font-bold text-white break-words">{{ organization.name }}</h1>
            <p class="text-slate-400 break-words">{{ organization.legal_name }}</p>
          </div>
          <Badge :variant="statusVariants[organization.status]">
            {{ statusNames[organization.status] }}
          </Badge>
        </div>

        <!-- Info Card -->
        <Card class="bg-slate-800 border-slate-700">
          <CardHeader>
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-500/10">
                <Building2 class="h-5 w-5 text-indigo-400" />
              </div>
              <CardTitle class="text-white">Информация об организации</CardTitle>
            </div>
          </CardHeader>
          <CardContent>
            <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div class="flex items-start gap-3">
                <FileText class="h-4 w-4 text-slate-500 mt-0.5" />
                <div>
                  <dt class="text-sm text-slate-400">ИНН</dt>
                  <dd class="text-white font-mono">{{ organization.inn }}</dd>
                </div>
              </div>
              <div class="flex items-start gap-3">
                <Globe class="h-4 w-4 text-slate-500 mt-0.5" />
                <div>
                  <dt class="text-sm text-slate-400">Страна</dt>
                  <dd class="text-white">{{ countryNames[organization.country] }}</dd>
                </div>
              </div>
              <div class="flex items-start gap-3">
                <Phone class="h-4 w-4 text-slate-500 mt-0.5" />
                <div>
                  <dt class="text-sm text-slate-400">Телефон</dt>
                  <dd class="text-white">{{ organization.phone }}</dd>
                </div>
              </div>
              <div class="flex items-start gap-3">
                <Mail class="h-4 w-4 text-slate-500 mt-0.5" />
                <div>
                  <dt class="text-sm text-slate-400">Email</dt>
                  <dd class="text-white">{{ organization.email }}</dd>
                </div>
              </div>
              <div class="flex items-start gap-3 sm:col-span-2">
                <MapPin class="h-4 w-4 text-slate-500 mt-0.5 shrink-0" />
                <div>
                  <dt class="text-sm text-slate-400">Адрес</dt>
                  <dd class="text-white break-words">{{ organization.address }}</dd>
                </div>
              </div>
              <div class="flex items-start gap-3">
                <Calendar class="h-4 w-4 text-slate-500 mt-0.5" />
                <div>
                  <dt class="text-sm text-slate-400">Дата регистрации</dt>
                  <dd class="text-white">{{ formatDate(organization.created_at) }}</dd>
                </div>
              </div>
            </dl>
          </CardContent>
        </Card>

        <!-- Members -->
        <Card class="bg-slate-800 border-slate-700">
          <CardHeader>
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-slate-700">
                <Users class="h-5 w-5 text-slate-400" />
              </div>
              <CardTitle class="text-white">Сотрудники</CardTitle>
            </div>
          </CardHeader>
          <CardContent class="p-0">
            <Table>
              <TableHeader>
                <TableRow class="border-slate-700 hover:bg-transparent">
                  <TableHead class="text-slate-300">Имя</TableHead>
                  <TableHead class="text-slate-300">Email</TableHead>
                  <TableHead class="text-slate-300">Телефон</TableHead>
                  <TableHead class="text-slate-300">Роль</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow
                  v-for="member in organization.members"
                  :key="member.id"
                  class="border-slate-700"
                >
                  <TableCell class="text-white font-medium">{{ member.name }}</TableCell>
                  <TableCell class="text-slate-300">{{ member.email }}</TableCell>
                  <TableCell class="text-slate-300">{{ member.phone }}</TableCell>
                  <TableCell>
                    <Badge variant="secondary" class="bg-slate-700 text-slate-300">
                      {{ roleNames[member.role] }}
                    </Badge>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <!-- Actions -->
        <div v-if="organization.status === 'pending'" class="flex gap-4">
          <Button
            class="flex-1 bg-green-600 hover:bg-green-500 text-white"
            :disabled="actionLoading"
            @click="handleApprove"
          >
            <Check class="h-4 w-4 mr-2" />
            {{ actionLoading ? 'Обработка...' : 'Одобрить' }}
          </Button>
          <Button
            variant="destructive"
            class="flex-1"
            :disabled="actionLoading"
            @click="showRejectModal = true"
          >
            <X class="h-4 w-4 mr-2" />
            Отклонить
          </Button>
        </div>
      </div>
    </main>

    <!-- Reject Modal -->
    <Dialog v-model:open="showRejectModal">
      <DialogContent class="bg-slate-800 border-slate-700 text-white sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="text-white">Отклонить организацию</DialogTitle>
          <DialogDescription class="text-slate-400">
            Укажите причину отклонения заявки
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label class="text-slate-200">Причина отклонения</Label>
          <Textarea
            v-model="rejectReason"
            rows="3"
            class="bg-slate-700 border-slate-600 text-white resize-none"
            placeholder="Укажите причину..."
          />
        </div>

        <DialogFooter>
          <Button
            variant="ghost"
            class="text-slate-400 hover:text-white"
            :disabled="actionLoading"
            @click="showRejectModal = false"
          >
            Отмена
          </Button>
          <Button
            variant="destructive"
            :disabled="!rejectReason.trim() || actionLoading"
            @click="handleReject"
          >
            {{ actionLoading ? 'Отклонение...' : 'Отклонить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
