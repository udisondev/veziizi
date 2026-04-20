<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { membersApi, type MemberProfile } from '@/api/members'
import {
  roleLabels,
  roleColors,
  statusLabels,
  statusColors,
} from '@/types/member'

// UI Components
import { Button } from '@/components/ui/button'
import { AppLink } from '@/components/ui/app-link'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

// Shared Components
import { DetailPageHeader } from '@/components/shared'

// Icons
import { MoreVertical, Pencil, AlertCircle } from 'lucide-vue-next'

const route = useRoute()
const auth = useAuthStore()

const member = ref<MemberProfile | null>(null)
const isLoading = ref(true)
const error = ref('')

// Block modal
const showBlockModal = ref(false)
const blockReason = ref('')
const blockLoading = ref(false)
const blockError = ref('')

// Unblock modal
const showUnblockModal = ref(false)
const unblockLoading = ref(false)
const unblockError = ref('')

// Edit modal
const showEditModal = ref(false)
const editName = ref('')
const editEmail = ref('')
const editPhone = ref('')
const editLoading = ref(false)
const editError = ref('')

// Permissions
const canManage = computed(() => {
  if (!member.value || !auth.organizationId) return false
  // User must be from the same organization
  if (member.value.organization_id !== auth.organizationId) return false
  // User must be owner or administrator
  return auth.role === 'owner' || auth.role === 'administrator'
})

const canBlock = computed(() => {
  if (!canManage.value || !member.value) return false
  // Cannot block owner
  if (member.value.role === 'owner') return false
  // Cannot block self
  if (member.value.id === auth.memberId) return false
  // Must be active
  return member.value.status === 'active'
})

const canUnblock = computed(() => {
  if (!canManage.value || !member.value) return false
  // Must be blocked
  return member.value.status === 'blocked'
})

const canEdit = computed(() => {
  if (!member.value || !auth.organizationId) return false
  // User must be from the same organization
  if (member.value.organization_id !== auth.organizationId) return false

  // Owner can only be edited by themselves
  if (member.value.role === 'owner') {
    return member.value.id === auth.memberId
  }

  // For non-owner members: manager can edit anyone, or member can edit themselves
  if (auth.role === 'owner' || auth.role === 'administrator') {
    return true
  }

  // Regular employee can only edit themselves
  return member.value.id === auth.memberId
})

const hasAnyAction = computed(() => canBlock.value || canUnblock.value || canEdit.value)

async function loadData() {
  isLoading.value = true
  error.value = ''
  try {
    const id = route.params.id as string
    member.value = await membersApi.getProfile(id)
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
  })
}

onMounted(() => {
  loadData()
})


watch(() => route.params.id, () => {
  loadData()
})

// Block actions
function openBlockModal() {
  blockReason.value = ''
  blockError.value = ''
  showBlockModal.value = true
}

function closeBlockModal() {
  showBlockModal.value = false
}

async function confirmBlock() {
  if (!member.value || !blockReason.value.trim()) {
    blockError.value = 'Укажите причину блокировки'
    return
  }

  blockLoading.value = true
  blockError.value = ''

  try {
    await membersApi.block(member.value.organization_id, member.value.id, blockReason.value.trim())
    // Optimistic update - сразу обновляем локальное состояние
    member.value.status = 'blocked'
    closeBlockModal()
  } catch (e: unknown) {
    blockError.value = e instanceof Error ? e.message : 'Не удалось заблокировать сотрудника'
  } finally {
    blockLoading.value = false
  }
}

// Unblock actions
function openUnblockModal() {
  unblockError.value = ''
  showUnblockModal.value = true
}

function closeUnblockModal() {
  showUnblockModal.value = false
}

async function confirmUnblock() {
  if (!member.value) return

  unblockLoading.value = true
  unblockError.value = ''

  try {
    await membersApi.unblock(member.value.organization_id, member.value.id)
    // Optimistic update - сразу обновляем локальное состояние
    member.value.status = 'active'
    closeUnblockModal()
  } catch (e: unknown) {
    unblockError.value = e instanceof Error ? e.message : 'Не удалось разблокировать сотрудника'
  } finally {
    unblockLoading.value = false
  }
}

// Edit actions
function openEditModal() {
  if (!member.value) return
  editName.value = member.value.name
  editEmail.value = member.value.email
  editPhone.value = member.value.phone || ''
  editError.value = ''
  showEditModal.value = true
}

function closeEditModal() {
  showEditModal.value = false
}

async function confirmEdit() {
  if (!member.value) return

  if (!editName.value.trim()) {
    editError.value = 'Укажите ФИО'
    return
  }
  if (!editEmail.value.trim()) {
    editError.value = 'Укажите Email'
    return
  }
  if (!editPhone.value.trim()) {
    editError.value = 'Укажите телефон'
    return
  }

  editLoading.value = true
  editError.value = ''

  try {
    await membersApi.updateInfo(
      member.value.organization_id,
      member.value.id,
      editName.value.trim(),
      editEmail.value.trim(),
      editPhone.value.trim()
    )
    // Optimistic update - сразу обновляем локальное состояние
    member.value.name = editName.value.trim()
    member.value.email = editEmail.value.trim()
    member.value.phone = editPhone.value.trim()
    closeEditModal()
  } catch (e: unknown) {
    editError.value = e instanceof Error ? e.message : 'Не удалось обновить данные сотрудника'
  } finally {
    editLoading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-background">
    <!-- Header -->
    <DetailPageHeader back-to="/" back-label="Назад" use-history>
      <template #actions>
        <!-- Десктоп/планшет: все кнопки отдельно -->
        <template v-if="member && hasAnyAction">
          <Button
            v-if="canEdit"
            variant="outline"
            size="sm"
            class="hidden md:inline-flex"
            data-tutorial="edit-member-btn"
            @click="openEditModal"
          >
            <Pencil class="mr-2 h-4 w-4" />
            Редактировать
          </Button>
          <Button
            v-if="canBlock"
            variant="outline"
            size="sm"
            class="hidden md:inline-flex text-destructive hover:text-destructive border-destructive/30 hover:border-destructive/60 hover:bg-destructive/5"
            data-tutorial="block-member-btn"
            @click="openBlockModal"
          >
            Заблокировать
          </Button>
          <Button
            v-if="canUnblock"
            variant="outline"
            size="sm"
            class="hidden md:inline-flex text-success hover:text-success border-success/30 hover:border-success/60 hover:bg-success/5"
            data-tutorial="unblock-member-btn"
            @click="openUnblockModal"
          >
            Разблокировать
          </Button>
        </template>

        <!-- Мобиле: бургер со всеми действиями -->
        <DropdownMenu v-if="member && hasAnyAction" class="md:hidden">
          <DropdownMenuTrigger as-child>
            <Button data-tutorial="member-actions" variant="ghost" size="icon" class="md:hidden">
              <MoreVertical class="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              v-if="canEdit"
              data-tutorial="edit-member-btn-mobile"
              @click="openEditModal"
            >
              Редактировать
            </DropdownMenuItem>
            <DropdownMenuItem
              v-if="canBlock"
              class="text-destructive focus:text-destructive"
              @click="openBlockModal"
            >
              Заблокировать
            </DropdownMenuItem>
            <DropdownMenuItem
              v-if="canUnblock"
              class="text-success focus:text-success"
              @click="openUnblockModal"
            >
              Разблокировать
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </template>
    </DetailPageHeader>

    <!-- Content -->
    <main class="max-w-4xl mx-auto px-4 py-6">
      <!-- Loading -->
      <div v-if="isLoading" class="text-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
        <div class="text-muted-foreground mt-2">Загрузка...</div>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 px-4 py-3 text-destructive">
        <AlertCircle class="h-4 w-4 shrink-0" />
        {{ error }}
        <button @click="loadData" class="ml-2 underline text-sm">Повторить</button>
      </div>

      <!-- Content -->
      <div v-else-if="member" class="space-y-6">
        <!-- Header Card -->
        <Card>
          <CardContent class="p-6">
            <h1 class="text-2xl font-bold text-foreground break-words">{{ member.name }}</h1>
            <div class="flex items-center gap-2 mt-2">
              <span :class="[roleColors[member.role], 'px-2 py-0.5 text-xs font-medium rounded-full']">
                {{ roleLabels[member.role] }}
              </span>
              <span :class="[statusColors[member.status], 'px-2 py-0.5 text-xs font-medium rounded-full']">
                {{ statusLabels[member.status] }}
              </span>
            </div>
          </CardContent>
        </Card>

        <!-- Details Card -->
        <Card>
          <CardContent class="p-6">
            <h2 class="text-lg font-semibold text-foreground mb-4">Информация</h2>
            <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <dt class="text-sm text-muted-foreground">ФИО</dt>
                <dd class="text-foreground font-medium break-words">{{ member.name }}</dd>
              </div>
              <div>
                <dt class="text-sm text-muted-foreground">Email</dt>
                <dd>
                  <a :href="`mailto:${member.email}`" class="app-link">{{ member.email }}</a>
                </dd>
              </div>
              <div v-if="member.phone">
                <dt class="text-sm text-muted-foreground">Телефон</dt>
                <dd>
                  <a :href="`tel:${member.phone}`" class="app-link">{{ member.phone }}</a>
                </dd>
              </div>
              <div>
                <dt class="text-sm text-muted-foreground">Организация</dt>
                <dd>
                  <AppLink
                    :to="{ name: 'organization-profile', params: { id: member.organization_id } }"
                    class="break-words"
                  >
                    {{ member.organization_name }}
                  </AppLink>
                </dd>
              </div>
              <div>
                <dt class="text-sm text-muted-foreground">Дата регистрации</dt>
                <dd class="text-foreground">{{ formatDate(member.created_at) }}</dd>
              </div>
            </dl>
          </CardContent>
        </Card>
      </div>
    </main>

    <!-- Block Modal -->
    <Dialog v-model:open="showBlockModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="text-destructive">Заблокировать сотрудника?</DialogTitle>
        </DialogHeader>

        <div
          v-if="blockError"
          class="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
        >
          <AlertCircle class="h-4 w-4 shrink-0" />
          {{ blockError }}
        </div>

        <div class="rounded-lg border border-warning/30 bg-warning/10 p-3">
          <p class="text-sm text-warning-foreground">
            Заблокированный сотрудник не сможет войти в систему и выполнять действия от имени организации.
          </p>
        </div>

        <div class="space-y-1">
          <p class="text-sm text-muted-foreground">
            Сотрудник: <strong class="text-foreground">{{ member?.name }}</strong>
          </p>
          <p class="text-sm text-muted-foreground">Email: {{ member?.email }}</p>
        </div>

        <div>
          <Label>
            Причина блокировки <span class="text-destructive">*</span>
          </Label>
          <Textarea
            v-model="blockReason"
            rows="3"
            placeholder="Укажите причину блокировки..."
          />
        </div>

        <div class="flex gap-3">
          <Button
            variant="outline"
            class="flex-1"
            :disabled="blockLoading"
            @click="closeBlockModal"
          >
            Отмена
          </Button>
          <Button
            variant="destructive"
            class="flex-1"
            :disabled="blockLoading || !blockReason.trim()"
            @click="confirmBlock"
          >
            {{ blockLoading ? 'Блокировка...' : 'Заблокировать' }}
          </Button>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Unblock Modal -->
    <Dialog v-model:open="showUnblockModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="text-success">Разблокировать сотрудника?</DialogTitle>
        </DialogHeader>

        <div
          v-if="unblockError"
          class="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
        >
          <AlertCircle class="h-4 w-4 shrink-0" />
          {{ unblockError }}
        </div>

        <div class="space-y-1">
          <p class="text-sm text-muted-foreground">
            Сотрудник: <strong class="text-foreground">{{ member?.name }}</strong>
          </p>
          <p class="text-sm text-muted-foreground">Email: {{ member?.email }}</p>
        </div>

        <p class="text-sm text-muted-foreground">
          После разблокировки сотрудник сможет снова войти в систему и выполнять действия.
        </p>

        <div class="flex gap-3">
          <Button
            variant="outline"
            class="flex-1"
            :disabled="unblockLoading"
            @click="closeUnblockModal"
          >
            Отмена
          </Button>
          <Button
            class="flex-1 bg-success text-success-foreground hover:bg-success/90"
            :disabled="unblockLoading"
            @click="confirmUnblock"
          >
            {{ unblockLoading ? 'Разблокировка...' : 'Разблокировать' }}
          </Button>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Edit Modal -->
    <Dialog v-model:open="showEditModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Редактирование данных сотрудника</DialogTitle>
        </DialogHeader>

        <div
          v-if="editError"
          class="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
        >
          <AlertCircle class="h-4 w-4 shrink-0" />
          {{ editError }}
        </div>

        <div class="space-y-4">
          <div>
            <Label>
              ФИО <span class="text-destructive">*</span>
            </Label>
            <Input
              v-model="editName"
              type="text"
              placeholder="Иванов Иван Иванович"
            />
          </div>

          <div>
            <Label>
              Email <span class="text-destructive">*</span>
            </Label>
            <Input
              v-model="editEmail"
              type="email"
              placeholder="example@mail.ru"
            />
          </div>

          <div>
            <Label>
              Телефон <span class="text-destructive">*</span>
            </Label>
            <Input
              v-model="editPhone"
              type="tel"
              placeholder="+7 (999) 123-45-67"
            />
          </div>
        </div>

        <div class="flex gap-3">
          <Button
            variant="outline"
            class="flex-1"
            :disabled="editLoading"
            @click="closeEditModal"
          >
            Отмена
          </Button>
          <Button
            class="flex-1"
            :disabled="editLoading || !editName.trim() || !editEmail.trim() || !editPhone.trim()"
            @click="confirmEdit"
          >
            {{ editLoading ? 'Сохранение...' : 'Сохранить' }}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
