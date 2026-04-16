<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'
import { membersApi } from '@/api/members'
import { freightRequestsApi } from '@/api/freightRequests'
import { historyApi } from '@/api/history'
import type { MemberListItem, MemberRole, MemberRoleFilter, MemberStatusFilter } from '@/types/member'
import {
  roleLabels,
  roleOptions,
  statusOptions,
} from '@/types/member'
import { memberStatusMapExtended as memberStatusMap } from '@/constants/statusMaps'
import { formatDateShort } from '@/utils/formatters'
import { logger } from '@/utils/logger'
import EventHistory from '@/components/EventHistory.vue'
import InvitationsTab from '@/components/members/InvitationsTab.vue'

// UI Components
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import { Tabs, TabsContent, TabsSlider } from '@/components/ui/tabs'
import { SelectField } from '@/components/ui/select-field'
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
  PageHeader,
  StatusBadge,
  LoadingSpinner,
  EmptyState,
  ErrorBanner,
  HighlightText,
  type TabItem,
} from '@/components/shared'

// Icons
import { Users, UserPlus, History } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { canManageInvitations, canViewHistory } = usePermissions()

// Tabs
const currentTab = ref('members')

// History loader
function loadOrganizationHistory(limit: number, offset: number) {
  if (!auth.organizationId) {
    return Promise.resolve({ items: [], total: 0 })
  }
  return historyApi.getOrganizationHistory(auth.organizationId, { limit, offset })
}

// Selection mode (for reassigning freight request responsible or carrier member)
const isSelectionMode = computed(() =>
  route.query.selectFor === 'freightRequest' || route.query.selectFor === 'carrierMember'
)
const freightRequestId = computed(() => route.query.frId as string | undefined)
const selectLoading = ref(false)

// Data
const members = ref<MemberListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)

// Filters
const searchQuery = ref('')
const roleFilter = ref<MemberRoleFilter>('all')
const statusFilter = ref<MemberStatusFilter>('all')

// Invitations tab ref
const invitationsTabRef = ref<InstanceType<typeof InvitationsTab> | null>(null)

// Template refs used in template via ref="..." (vue-tsc false positive workaround)
void invitationsTabRef

// Computed
const filteredMembers = computed(() => {
  let result = members.value

  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(
      (m) =>
        m.name.toLowerCase().includes(q) ||
        m.email.toLowerCase().includes(q) ||
        (m.phone && m.phone.includes(q))
    )
  }

  if (roleFilter.value !== 'all') {
    result = result.filter((m) => m.role === roleFilter.value)
  }

  if (statusFilter.value !== 'all') {
    result = result.filter((m) => m.status === statusFilter.value)
  }

  // Sort: owner first, then by date
  return result.sort((a, b) => {
    if (a.role === 'owner') return -1
    if (b.role === 'owner') return 1
    return new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  })
})

const hasActiveFilters = computed(
  () => searchQuery.value.trim() !== '' || roleFilter.value !== 'all' || statusFilter.value !== 'all'
)


const tabItems = computed((): TabItem[] => {
  const items: TabItem[] = [
    { value: 'members', label: 'Сотрудники', icon: Users },
  ]
  if (canManageInvitations.value) {
    items.push({ value: 'invitations', label: 'Приглашения', icon: UserPlus })
  }
  if (canViewHistory.value) {
    items.push({ value: 'history', label: 'История', icon: History, separator: true })
  }
  return items
})

// Load data
async function loadMembers() {
  if (!auth.organizationId) return

  isLoading.value = true
  error.value = null

  try {
    members.value = await membersApi.listByOrganization(auth.organizationId)
  } catch (e) {
    error.value = 'Не удалось загрузить список сотрудников'
    logger.error('Failed to load members', e)
  } finally {
    isLoading.value = false
  }
}

function resetAllFilters() {
  searchQuery.value = ''
  roleFilter.value = 'all'
  statusFilter.value = 'all'
}

// Navigation to member detail
function goToMember(member: MemberListItem) {
  if (isSelectionMode.value) return
  router.push({ name: 'member-profile', params: { id: member.id } })
}

// Selection mode handlers
async function selectMember(member: MemberListItem) {
  if (!isSelectionMode.value || !freightRequestId.value) return
  if (member.status !== 'active') return

  selectLoading.value = true
  error.value = null

  try {
    const selectFor = route.query.selectFor
    if (selectFor === 'carrierMember') {
      await freightRequestsApi.reassignCarrier(freightRequestId.value, member.id)
    } else {
      await freightRequestsApi.reassign(freightRequestId.value, member.id)
    }
    router.push(`/freight-requests/${freightRequestId.value}`)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Не удалось назначить ответственного'
  } finally {
    selectLoading.value = false
  }
}

function cancelSelection() {
  if (freightRequestId.value) {
    router.push(`/freight-requests/${freightRequestId.value}`)
  } else {
    router.back()
  }
}

function getRoleBadgeVariant(role: MemberRole): 'default' | 'secondary' | 'outline' {
  if (role === 'owner') return 'default'
  if (role === 'administrator') return 'secondary'
  return 'outline'
}

onMounted(() => {
  loadMembers()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <!-- Header -->
    <PageHeader
      :title="isSelectionMode ? 'Выберите ответственного' : 'Штат'"
      class="mb-6"
    >
      <template #description v-if="isSelectionMode">
        Нажмите на сотрудника для назначения
      </template>
      <template #actions>
        <Button
          v-if="isSelectionMode"
          variant="outline"
          @click="cancelSelection"
        >
          Отмена
        </Button>
      </template>
    </PageHeader>

    <!-- Tabs -->
    <Tabs v-if="(canManageInvitations || canViewHistory) && !isSelectionMode" v-model="currentTab" class="space-y-6">
      <!-- Tab selector with slider -->
      <TabsSlider v-model="currentTab" :items="tabItems" />

      <!-- Members Tab -->
      <TabsContent value="members">
        <!-- Inline Filters -->
        <div class="flex flex-col sm:flex-row gap-2 mb-4">
          <Input
            v-model="searchQuery"
            placeholder="ФИО, email или телефон"
            class="sm:max-w-xs"
          />
          <div class="sm:w-44">
            <SelectField
              v-model="roleFilter"
              :options="roleOptions"
              sheet-label="Роль"
            />
          </div>
          <div class="sm:w-48">
            <SelectField
              v-model="statusFilter"
              :options="statusOptions"
              sheet-label="Статус"
            />
          </div>
          <Button v-if="hasActiveFilters" variant="ghost" size="sm" @click="resetAllFilters">
            Сбросить
          </Button>
        </div>

        <LoadingSpinner v-if="isLoading" text="Загрузка сотрудников..." />

        <ErrorBanner
          v-else-if="error"
          :message="error"
          @retry="loadMembers"
        />

        <EmptyState
          v-else-if="filteredMembers.length === 0"
          :icon="Users"
          :title="hasActiveFilters ? 'Нет сотрудников по фильтрам' : 'Сотрудников пока нет'"
          :description="hasActiveFilters ? 'Попробуйте изменить параметры фильтрации' : 'Пригласите первого сотрудника'"
        />

        <template v-else>
          <!-- Mobile Cards -->
          <div class="sm:hidden space-y-3">
            <Card
              v-for="member in filteredMembers"
              :key="member.id"
              :class="cn(
                'cursor-pointer transition-shadow',
                isSelectionMode && member.status === 'active' && 'hover:shadow-md',
                isSelectionMode && member.status !== 'active' && 'opacity-50 cursor-not-allowed',
                !isSelectionMode && 'hover:shadow-md',
                selectLoading && 'pointer-events-none'
              )"
              @click="isSelectionMode ? selectMember(member) : goToMember(member)"
            >
              <CardContent class="p-4">
                <div class="flex items-start justify-between gap-2">
                  <div class="min-w-0 flex-1">
                    <div class="font-medium text-foreground truncate">
                      <HighlightText :text="member.name" :query="searchQuery" />
                      <span v-if="member.id === auth.memberId" class="text-xs text-muted-foreground">(вы)</span>
                    </div>
                    <div class="text-sm text-muted-foreground truncate">
                      <HighlightText :text="member.email" :query="searchQuery" />
                    </div>
                    <div v-if="member.phone" class="text-sm text-muted-foreground">
                      <HighlightText :text="member.phone" :query="searchQuery" />
                    </div>
                  </div>
                  <div class="flex flex-col items-end gap-1">
                    <Badge :variant="getRoleBadgeVariant(member.role)">
                      {{ roleLabels[member.role] }}
                    </Badge>
                    <StatusBadge :status="member.status" :status-map="memberStatusMap" />
                  </div>
                </div>
                <div class="mt-2 text-xs text-muted-foreground">
                  Добавлен {{ formatDateShort(member.created_at) }}
                </div>
              </CardContent>
            </Card>
          </div>

          <!-- Desktop Table -->
          <Card class="hidden sm:block">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Дата</TableHead>
                  <TableHead>ФИО</TableHead>
                  <TableHead>Телефон</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Роль</TableHead>
                  <TableHead>Статус</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow
                  v-for="member in filteredMembers"
                  :key="member.id"
                  :class="cn(
                    'cursor-pointer',
                    isSelectionMode && member.status === 'active' && 'hover:bg-primary/5',
                    isSelectionMode && member.status !== 'active' && 'opacity-50 cursor-not-allowed',
                    selectLoading && 'pointer-events-none'
                  )"
                  @click="isSelectionMode ? selectMember(member) : goToMember(member)"
                >
                  <TableCell class="text-muted-foreground">
                    {{ formatDateShort(member.created_at) }}
                  </TableCell>
                  <TableCell>
                    <div class="font-medium">
                      <HighlightText :text="member.name" :query="searchQuery" />
                    </div>
                    <div v-if="member.id === auth.memberId" class="text-xs text-muted-foreground">
                      (это вы)
                    </div>
                  </TableCell>
                  <TableCell class="text-muted-foreground">
                    <HighlightText v-if="member.phone" :text="member.phone" :query="searchQuery" />
                    <span v-else>—</span>
                  </TableCell>
                  <TableCell class="text-muted-foreground">
                    <HighlightText :text="member.email" :query="searchQuery" />
                  </TableCell>
                  <TableCell>
                    <Badge :variant="getRoleBadgeVariant(member.role)">
                      {{ roleLabels[member.role] }}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <StatusBadge :status="member.status" :status-map="memberStatusMap" />
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </Card>
        </template>
      </TabsContent>

      <!-- Invitations Tab -->
      <TabsContent value="invitations">
        <InvitationsTab ref="invitationsTabRef" />
      </TabsContent>

      <!-- History Tab -->
      <TabsContent value="history">
        <EventHistory :load-fn="loadOrganizationHistory" />
      </TabsContent>
    </Tabs>

    <!-- Members list when no tabs (selection mode or no permissions) -->
    <template v-if="(!canManageInvitations && !canViewHistory) || isSelectionMode">
      <LoadingSpinner v-if="isLoading" text="Загрузка сотрудников..." />

      <ErrorBanner
        v-else-if="error"
        :message="error"
        @retry="loadMembers"
      />

      <EmptyState
        v-else-if="filteredMembers.length === 0"
        :icon="Users"
        :title="hasActiveFilters ? 'Нет сотрудников по фильтрам' : 'Сотрудников пока нет'"
        :description="hasActiveFilters ? 'Попробуйте изменить параметры фильтрации' : ''"
      />

      <div v-else class="space-y-3">
        <Card
          v-for="member in filteredMembers"
          :key="member.id"
          :class="cn(
            'cursor-pointer transition-shadow',
            isSelectionMode && member.status === 'active' && 'hover:shadow-md',
            isSelectionMode && member.status !== 'active' && 'opacity-50 cursor-not-allowed',
            !isSelectionMode && 'hover:shadow-md',
            selectLoading && 'pointer-events-none'
          )"
          @click="isSelectionMode ? selectMember(member) : goToMember(member)"
        >
          <CardContent class="p-4">
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0 flex-1">
                <div class="font-medium text-foreground truncate">
                  {{ member.name }}
                  <span v-if="member.id === auth.memberId" class="text-xs text-muted-foreground">(вы)</span>
                </div>
                <div class="text-sm text-muted-foreground truncate">{{ member.email }}</div>
              </div>
              <div class="flex flex-col items-end gap-1">
                <Badge :variant="getRoleBadgeVariant(member.role)">
                  {{ roleLabels[member.role] }}
                </Badge>
                <StatusBadge :status="member.status" :status-map="memberStatusMap" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </template>

  </div>
</template>
