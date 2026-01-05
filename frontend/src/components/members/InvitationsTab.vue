<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { vMaska } from 'maska/vue'
import { invitationsApi } from '@/api/invitations'
import type { InvitationListItem, InvitationStatus, InvitationRole } from '@/types/invitation'
import { invitationStatusMap } from '@/constants/statusMaps'
import { formatDateTime } from '@/utils/formatters'
import { logger } from '@/utils/logger'

// UI Components
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

// Shared Components
import {
  StatusBadge,
  LoadingSpinner,
  EmptyState,
  ErrorBanner,
  ConfirmDialog,
  FilterSheet,
} from '@/components/shared'

// Icons
import { Mail, Plus, Copy, Check, Clock, AlertCircle } from 'lucide-vue-next'

const auth = useAuthStore()

// Data
const invitations = ref<InvitationListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)

// Filters
const showFilters = ref(false)
type InvitationStatusFilter = InvitationStatus | 'all'
const statusFilter = ref<InvitationStatusFilter>('all')
const tempStatus = ref<InvitationStatusFilter>('all')

const statusOptions: { value: InvitationStatusFilter; label: string }[] = [
  { value: 'all', label: 'Все статусы' },
  { value: 'pending', label: 'Ожидают' },
  { value: 'accepted', label: 'Приняты' },
  { value: 'expired', label: 'Истекли' },
  { value: 'cancelled', label: 'Отменены' },
]

const roleOptions: { value: InvitationRole; label: string }[] = [
  { value: 'employee', label: 'Сотрудник' },
  { value: 'administrator', label: 'Администратор' },
]

// Invitation form
const showForm = ref(false)
const isSubmitting = ref(false)
const formError = ref<string | null>(null)
const createdToken = ref<string | null>(null)
const copied = ref(false)
const phoneMask = '+7 (###) ###-##-##'
const phonePlaceholder = '+7 (999) 999-99-99'

const form = ref({
  email: '',
  role: 'employee' as InvitationRole,
  name: '',
  phone: '',
})

// Cancel invitation
const cancellingId = ref<string | null>(null)
const showCancelModal = ref(false)
const cancellingInvitation = ref<InvitationListItem | null>(null)
const cancelError = ref<string | null>(null)

// Computed
const hasActiveFilters = computed(() => statusFilter.value !== 'all')
const activeFiltersCount = computed(() => (statusFilter.value !== 'all' ? 1 : 0))

// Load data
async function loadData() {
  if (!auth.organizationId) return

  isLoading.value = true
  error.value = null

  try {
    const status = statusFilter.value !== 'all' ? (statusFilter.value as InvitationStatus) : undefined
    const response = await invitationsApi.list(auth.organizationId, status)
    invitations.value = response.items ?? []
  } catch (e) {
    error.value = 'Не удалось загрузить приглашения'
    logger.error('Failed to load invitations', e)
  } finally {
    isLoading.value = false
  }
}

// CRUD
async function createInvitation() {
  if (!auth.organizationId) return

  isSubmitting.value = true
  formError.value = null
  createdToken.value = null

  try {
    const response = await invitationsApi.create(auth.organizationId, {
      email: form.value.email,
      role: form.value.role,
      name: form.value.name || undefined,
      phone: form.value.phone || undefined,
    })

    createdToken.value = response.token
    form.value = { email: '', role: 'employee', name: '', phone: '' }
    await loadData()
  } catch (e: unknown) {
    formError.value = e instanceof Error ? e.message : 'Не удалось создать приглашение'
    logger.error('Failed to create invitation', e)
  } finally {
    isSubmitting.value = false
  }
}

function closeForm() {
  showForm.value = false
  createdToken.value = null
  formError.value = null
  copied.value = false
}

function openCancelModal(item: InvitationListItem) {
  cancellingInvitation.value = item
  cancelError.value = null
  showCancelModal.value = true
}

function closeCancelModal() {
  showCancelModal.value = false
  cancellingInvitation.value = null
  cancelError.value = null
}

async function confirmCancel() {
  if (!auth.organizationId || !cancellingInvitation.value) return

  cancellingId.value = cancellingInvitation.value.id
  cancelError.value = null

  try {
    await invitationsApi.cancel(auth.organizationId, cancellingInvitation.value.id)
    const item = invitations.value.find((i) => i.id === cancellingInvitation.value!.id)
    if (item) {
      item.status = 'cancelled'
    }
    closeCancelModal()
  } catch (e: unknown) {
    logger.error('Failed to cancel invitation', e)
    cancelError.value = e instanceof Error ? e.message : 'Не удалось отменить приглашение'
  } finally {
    cancellingId.value = null
  }
}

// Helpers
function getRoleLabel(role: string): string {
  switch (role) {
    case 'employee':
      return 'Сотрудник'
    case 'administrator':
      return 'Администратор'
    default:
      return role
  }
}

function getInvitationUrl(token: string): string {
  return `${window.location.origin}/invitations/${token}`
}

async function copyToClipboard(text: string) {
  await navigator.clipboard.writeText(text)
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 2000)
}

// Filter functions
function openFilters() {
  tempStatus.value = statusFilter.value
  showFilters.value = true
}

function applyFilters() {
  statusFilter.value = tempStatus.value
  loadData()
  showFilters.value = false
}

function resetFilters() {
  tempStatus.value = 'all'
}

function resetAllFilters() {
  statusFilter.value = 'all'
  loadData()
}

// Expose for parent
defineExpose({
  loadData,
  showForm,
})

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="space-y-4">
    <!-- Toolbar -->
    <div class="flex items-center justify-between gap-2">
      <FilterSheet
        v-model:open="showFilters"
        :active-filters-count="activeFiltersCount"
        description="Фильтрация приглашений"
        @open="openFilters"
        @apply="applyFilters"
        @reset="resetFilters"
      >
        <div class="space-y-2">
          <Label>Статус</Label>
          <Select v-model="tempStatus">
            <SelectTrigger>
              <SelectValue placeholder="Все статусы" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="opt in statusOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </FilterSheet>

      <Button data-tutorial="invite-btn" @click="showForm = true">
        <Plus class="mr-2 h-4 w-4" />
        Пригласить
      </Button>
    </div>

    <!-- Active filters -->
    <Card v-if="hasActiveFilters" class="border-primary/20 bg-primary/5">
      <CardContent class="flex items-center justify-between py-3">
        <span class="text-sm text-primary">
          Статус: {{ statusOptions.find((o) => o.value === statusFilter)?.label }}
        </span>
        <Button variant="ghost" size="sm" @click="resetAllFilters"> Сбросить </Button>
      </CardContent>
    </Card>

    <!-- Loading -->
    <LoadingSpinner v-if="isLoading" text="Загрузка приглашений..." />

    <!-- Error -->
    <ErrorBanner v-else-if="error" :message="error" @retry="loadData" />

    <!-- Empty -->
    <EmptyState
      v-else-if="invitations.length === 0"
      :icon="Mail"
      :title="hasActiveFilters ? 'Нет приглашений по фильтрам' : 'Приглашений пока нет'"
      :description="hasActiveFilters ? 'Попробуйте изменить параметры фильтрации' : 'Создайте приглашение для нового сотрудника'"
    >
      <template #action>
        <Button @click="showForm = true">
          <Plus class="mr-2 h-4 w-4" />
          Создать приглашение
        </Button>
      </template>
    </EmptyState>

    <!-- Content -->
    <template v-else>
      <!-- Mobile Cards -->
      <div class="sm:hidden space-y-3">
        <Card v-for="item in invitations" :key="item.id">
          <CardContent class="p-4">
            <div class="flex items-start justify-between gap-2 mb-2">
              <div class="min-w-0 flex-1">
                <div class="font-medium text-foreground truncate">{{ item.email }}</div>
                <div v-if="item.name" class="text-sm text-muted-foreground truncate">{{ item.name }}</div>
                <div v-if="item.phone" class="text-sm text-muted-foreground">{{ item.phone }}</div>
              </div>
              <div class="flex flex-col items-end gap-1">
                <Badge variant="outline">
                  {{ getRoleLabel(item.role) }}
                </Badge>
                <StatusBadge :status="item.status" :status-map="invitationStatusMap" />
              </div>
            </div>
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-1 text-xs text-muted-foreground">
                <Clock class="h-3 w-3" />
                {{ formatDateTime(item.expires_at) }}
              </div>
              <Button
                v-if="item.status === 'pending'"
                variant="ghost"
                size="sm"
                class="text-destructive hover:text-destructive"
                @click="openCancelModal(item)"
              >
                Отменить
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- Desktop Table -->
      <Card class="hidden sm:block">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Email</TableHead>
              <TableHead>Роль</TableHead>
              <TableHead>ФИО</TableHead>
              <TableHead>Телефон</TableHead>
              <TableHead>Статус</TableHead>
              <TableHead>Истекает</TableHead>
              <TableHead>Действия</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="item in invitations" :key="item.id">
              <TableCell class="font-medium">{{ item.email }}</TableCell>
              <TableCell>{{ getRoleLabel(item.role) }}</TableCell>
              <TableCell class="text-muted-foreground">{{ item.name || '—' }}</TableCell>
              <TableCell class="text-muted-foreground">{{ item.phone || '—' }}</TableCell>
              <TableCell>
                <StatusBadge :status="item.status" :status-map="invitationStatusMap" />
              </TableCell>
              <TableCell class="text-muted-foreground">
                {{ formatDateTime(item.expires_at) }}
              </TableCell>
              <TableCell>
                <Button
                  v-if="item.status === 'pending'"
                  variant="ghost"
                  size="sm"
                  class="text-destructive hover:text-destructive"
                  @click="openCancelModal(item)"
                >
                  Отменить
                </Button>
                <span v-else class="text-muted-foreground">—</span>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </Card>
    </template>

    <!-- Create Dialog -->
    <Dialog v-model:open="showForm">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Новое приглашение</DialogTitle>
          <DialogDescription> Отправьте приглашение новому сотруднику </DialogDescription>
        </DialogHeader>

        <!-- Success -->
        <div v-if="createdToken" class="space-y-4">
          <div class="rounded-lg border border-success/50 bg-success/10 p-4">
            <p class="text-success font-medium mb-2">Приглашение создано!</p>
            <p class="text-sm text-muted-foreground mb-2">Ссылка для приглашения:</p>
            <div class="flex items-center gap-2">
              <Input :model-value="getInvitationUrl(createdToken)" readonly class="flex-1 text-sm" />
              <Button size="icon" variant="outline" @click="copyToClipboard(getInvitationUrl(createdToken))">
                <Check v-if="copied" class="h-4 w-4 text-success" />
                <Copy v-else class="h-4 w-4" />
              </Button>
            </div>
          </div>
          <Button variant="outline" class="w-full" @click="closeForm"> Закрыть </Button>
        </div>

        <!-- Form -->
        <form v-else class="space-y-4" @submit.prevent="createInvitation">
          <div
            v-if="formError"
            class="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
          >
            <AlertCircle class="h-4 w-4 shrink-0" />
            {{ formError }}
          </div>

          <div class="space-y-2">
            <Label for="inv-email"> Email <span class="text-destructive">*</span> </Label>
            <Input id="inv-email" v-model="form.email" type="email" required placeholder="user@example.com" />
          </div>

          <div class="space-y-2">
            <Label for="inv-role"> Роль <span class="text-destructive">*</span> </Label>
            <Select v-model="form.role">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="opt in roleOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="space-y-2">
            <Label for="inv-name">
              ФИО
              <span class="text-muted-foreground font-normal">(опционально)</span>
            </Label>
            <Input id="inv-name" v-model="form.name" placeholder="Иванов Иван Иванович" />
            <p class="text-xs text-muted-foreground">Если заполнить, приглашённый не сможет изменить</p>
          </div>

          <div class="space-y-2">
            <Label for="inv-phone">
              Телефон
              <span class="text-muted-foreground font-normal">(опционально)</span>
            </Label>
            <Input
              id="inv-phone"
              v-model="form.phone"
              v-maska
              :data-maska="phoneMask"
              type="tel"
              inputmode="tel"
              :placeholder="phonePlaceholder"
            />
            <p class="text-xs text-muted-foreground">Если заполнить, приглашённый не сможет изменить</p>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" @click="closeForm"> Отмена </Button>
            <Button type="submit" :disabled="isSubmitting">
              {{ isSubmitting ? 'Создание...' : 'Создать' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Cancel Dialog -->
    <ConfirmDialog
      :open="showCancelModal"
      title="Отменить приглашение?"
      :description="`Вы уверены, что хотите отменить приглашение для ${cancellingInvitation?.email}? Пользователь не сможет принять это приглашение.`"
      confirm-text="Отменить приглашение"
      confirm-variant="destructive"
      :loading="cancellingId !== null"
      @confirm="confirmCancel"
      @cancel="closeCancelModal"
    />
  </div>
</template>
