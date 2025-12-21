<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'

// Shared Components
import { PageHeader } from '@/components/shared'

// Icons
import { Building2, AlertCircle } from 'lucide-vue-next'

const auth = useAuthStore()

// Organization status labels
const statusLabels: Record<string, string> = {
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
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <PageHeader title="Настройки организации" class="mb-6" />

    <template v-if="auth.organization">
      <!-- Organization Info Card -->
      <Card class="max-w-xl">
        <CardHeader>
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Building2 class="h-5 w-5 text-primary" />
            </div>
            <div>
              <CardTitle class="text-lg">Информация об организации</CardTitle>
              <CardDescription>Основные данные вашей организации</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">Статус</span>
            <Badge :variant="statusVariants[auth.organization.status]">
              {{ statusLabels[auth.organization.status] }}
            </Badge>
          </div>
          <Separator />
          <div>
            <p class="text-sm text-muted-foreground mb-1">Название</p>
            <p class="font-medium">{{ auth.organization.name }}</p>
          </div>
        </CardContent>
      </Card>

      <!-- Info Banner -->
      <Card class="mt-6 max-w-xl border-blue-500/20 bg-blue-500/5">
        <CardContent class="flex items-start gap-3 py-4">
          <AlertCircle class="h-5 w-5 text-blue-500 shrink-0 mt-0.5" />
          <div>
            <p class="text-sm font-medium text-blue-700 dark:text-blue-400">Редактирование недоступно</p>
            <p class="text-sm text-muted-foreground mt-1">
              Для изменения данных организации обратитесь в службу поддержки.
            </p>
          </div>
        </CardContent>
      </Card>
    </template>

    <!-- No organization data -->
    <Card v-else class="max-w-xl">
      <CardContent class="py-12 text-center">
        <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-muted mb-4">
          <Building2 class="h-8 w-8 text-muted-foreground" />
        </div>
        <p class="text-muted-foreground">Данные организации не загружены</p>
        <Button variant="outline" class="mt-4" @click="auth.fetchMe()">
          Обновить
        </Button>
      </CardContent>
    </Card>
  </div>
</template>
