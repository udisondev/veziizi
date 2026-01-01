<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'

// UI Components
import { Button } from '@/components/ui/button'
import NotificationBell from '@/components/notifications/NotificationBell.vue'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Separator } from '@/components/ui/separator'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

// Icons
import {
  Menu,
  Package,
  ClipboardList,
  HandCoins,
  Bell,
  Users,
  Settings,
  User,
  LogOut,
  Building2,
  HelpCircle,
} from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const { canManageMembers } = usePermissions()

const isMenuOpen = ref(false)

const menuItems = computed(() => {
  const items = [
    { to: '/', label: 'Заявки', icon: Package },
    { to: '/orders', label: 'Заказы', icon: ClipboardList },
    { to: '/my-offers', label: 'Предложения', icon: HandCoins },
    { to: '/subscriptions', label: 'Рассылка', icon: Bell },
    { to: '/members', label: 'Штат', icon: Users },
  ]

  if (canManageMembers.value) {
    items.push(
      { to: '/organization/settings', label: 'Настройки', icon: Settings },
    )
  }

  items.push({ to: '/profile', label: 'Профиль', icon: User })

  return items
})

function navigate(to: string) {
  router.push(to)
  isMenuOpen.value = false
}

async function logout() {
  await auth.logout()
  router.push('/login')
}

const userInitial = computed(() => {
  return auth.name?.charAt(0).toUpperCase() || '?'
})
</script>

<template>
  <header class="bg-card border-b sticky top-0 z-50">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex items-center justify-between h-14">
        <!-- Left: Menu button + Title -->
        <div class="flex items-center gap-3">
          <!-- Mobile menu (Sheet) -->
          <Sheet v-model:open="isMenuOpen">
            <SheetTrigger as-child>
              <Button variant="ghost" size="icon" class="md:hidden">
                <Menu class="h-5 w-5" />
              </Button>
            </SheetTrigger>
            <SheetContent side="left" class="w-72">
              <SheetHeader>
                <SheetTitle class="text-left">Меню</SheetTitle>
              </SheetHeader>
              <nav class="mt-6 space-y-1">
                <button
                  v-for="item in menuItems"
                  :key="item.to"
                  @click="navigate(item.to)"
                  :class="[
                    'w-full text-left px-3 py-2 rounded-md text-sm flex items-center gap-3 transition-colors',
                    route.path === item.to
                      ? 'bg-primary/10 text-primary'
                      : 'text-foreground hover:bg-muted'
                  ]"
                >
                  <component :is="item.icon" class="h-5 w-5" />
                  <span>{{ item.label }}</span>
                </button>

                <Separator class="my-3" />

                <button
                  @click="logout"
                  class="w-full text-left px-3 py-2 rounded-md text-sm text-destructive hover:bg-destructive/10 flex items-center gap-3 transition-colors"
                >
                  <LogOut class="h-5 w-5" />
                  <span>Выйти</span>
                </button>
              </nav>
            </SheetContent>
          </Sheet>

          <!-- Desktop navigation -->
          <nav class="hidden md:flex items-center gap-1">
            <router-link
              v-for="item in menuItems.slice(0, 5)"
              :key="item.to"
              :to="item.to"
              :class="[
                'px-3 py-2 rounded-md text-sm font-medium flex items-center gap-2 transition-colors',
                route.path === item.to
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted'
              ]"
            >
              <component :is="item.icon" class="h-4 w-4" />
              <span>{{ item.label }}</span>
            </router-link>
          </nav>

          <!-- Logo/Title -->
          <router-link to="/" class="text-lg font-semibold text-foreground md:hidden">
            Veziizi
          </router-link>
        </div>

        <!-- Right: Organization + Notifications + User menu -->
        <div class="flex items-center gap-2">
          <!-- Organization name (desktop) -->
          <div class="hidden sm:flex items-center gap-2 text-sm text-muted-foreground mr-1">
            <Building2 class="h-4 w-4" />
            <span class="max-w-40 truncate">{{ auth.organization?.name }}</span>
          </div>

          <!-- Support -->
          <Button variant="ghost" size="icon" @click="router.push('/support')" title="Поддержка">
            <HelpCircle class="h-5 w-5" />
          </Button>

          <!-- Notifications -->
          <NotificationBell />

          <!-- User dropdown -->
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" class="relative h-9 w-9 rounded-full">
                <Avatar class="h-9 w-9">
                  <AvatarFallback class="bg-primary/10 text-primary">
                    {{ userInitial }}
                  </AvatarFallback>
                </Avatar>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="w-56">
              <DropdownMenuLabel>
                <div class="flex flex-col space-y-1">
                  <p class="text-sm font-medium">{{ auth.name }}</p>
                  <p class="text-xs text-muted-foreground">{{ auth.email }}</p>
                </div>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem @click="router.push('/profile')">
                <User class="mr-2 h-4 w-4" />
                Профиль
              </DropdownMenuItem>
              <DropdownMenuItem
                v-if="canManageMembers"
                @click="router.push('/organization/settings')"
              >
                <Settings class="mr-2 h-4 w-4" />
                Настройки
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                class="text-destructive focus:text-destructive"
                @click="logout"
              >
                <LogOut class="mr-2 h-4 w-4" />
                Выйти
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </div>
  </header>
</template>
