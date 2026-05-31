<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingStore } from '@/stores/onboarding'
import { useFreightFiltersStore } from '@/stores/freightFilters'
import { usePermissions } from '@/composables/usePermissions'
import { useBreakpoint } from '@/composables/useBreakpoint'
import { tutorialBus } from '@/sandbox/events'
import logoUrl from '@/assets/logo.svg'

// UI Components
import { Button } from '@/components/ui/button'
import NotificationBell from '@/components/notifications/NotificationBell.vue'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Separator } from '@/components/ui/separator'
import BottomSheet from '@/components/shared/BottomSheet.vue'
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
  HandCoins,
  Bell,
  Users,
  Settings,
  User,
  LogOut,
  Building2,
  HelpCircle,
  Package,
  LayoutDashboard,
  Truck,
  Bus,
} from 'lucide-vue-next'
import Tooltip from '@/components/ui/tooltip/Tooltip.vue'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const onboarding = useOnboardingStore()
const filtersStore = useFreightFiltersStore()
const { canManageMembers } = usePermissions()

const { isMobile } = useBreakpoint()
const isMenuOpen = ref(false)
const isProfileOpen = ref(false)

// Эмит события при открытии/закрытии мобильного меню для туториала
watch(isMenuOpen, (newValue) => {
  if (onboarding.isSandboxMode) {
    tutorialBus.emit(newValue ? 'menu:opened' : 'menu:closed')
  }
})

const menuItems = computed(() => {
  const items = [
    { to: '/', label: 'Дашборд', icon: LayoutDashboard },
    { to: '/requests', label: 'Заявки', icon: Package },
    { to: '/my-offers', label: 'Предложения', icon: HandCoins },
    { to: '/subscriptions', label: 'Рассылка', icon: Bell },
    { to: '/transport', label: 'Транспорт', icon: Bus },
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
  if (to === '/requests') filtersStore.resetFilters()
  router.push(to)
  isMenuOpen.value = false
}

function handleNavClick(to: string) {
  if (to === '/requests') filtersStore.resetFilters()
  if (to === '/requests' && onboarding.isSandboxMode) tutorialBus.emit('nav:requestsClicked')
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
  <header class="bg-slate-900 border-b border-slate-800 sticky top-0 z-50">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex items-center justify-between h-14">
        <!-- Left: Menu button + Title -->
        <div class="flex items-center gap-3">
          <!-- Mobile menu -->
          <Button variant="ghost" size="icon" class="lg:hidden text-slate-300 hover:text-white hover:bg-slate-800" data-tutorial="mobile-menu-btn" @click="isMenuOpen = true">
            <Menu class="h-5 w-5" />
          </Button>

          <!-- Logo -->
          <router-link to="/" class="shrink-0">
            <img :src="logoUrl" alt="ВезиИзи" class="h-9 w-auto" />
          </router-link>
          <BottomSheet v-model="isMenuOpen" label="Меню">
            <nav class="px-2 pb-4 space-y-1">
              <button
                v-for="item in menuItems"
                :key="item.to"
                @click="navigate(item.to)"
                :data-tutorial="
                  item.to === '/requests' ? 'mobile-nav-requests' :
                  item.to === '/my-offers' ? 'mobile-nav-my-offers' :
                  item.to === '/subscriptions' ? 'mobile-nav-subscriptions' :
                  item.to === '/members' ? 'mobile-nav-members' : undefined
                "
                :class="[
                  'w-full text-left px-3 py-3 rounded-md text-base flex items-center gap-3 transition-colors',
                  route.path === item.to
                    ? 'bg-primary/10 text-primary'
                    : 'text-foreground hover:bg-muted'
                ]"
              >
                <component :is="item.icon" class="h-5 w-5" />
                <span>{{ item.label }}</span>
              </button>

              <Separator class="my-2" />

              <button
                @click="logout"
                class="w-full text-left px-3 py-3 rounded-md text-base text-destructive hover:bg-destructive/10 flex items-center gap-3 transition-colors"
              >
                <LogOut class="h-5 w-5" />
                <span>Выйти</span>
              </button>
            </nav>
          </BottomSheet>

          <!-- Desktop navigation -->
          <nav class="hidden lg:flex items-center gap-1 ml-3">
            <router-link
              v-for="item in menuItems.slice(0, 5)"
              :key="item.to"
              :to="item.to"
              :data-tutorial="
                item.to === '/requests' ? 'nav-requests' :
                item.to === '/my-offers' ? 'nav-my-offers' :
                item.to === '/subscriptions' ? 'nav-subscriptions' :
                item.to === '/members' ? 'nav-members' : undefined
              "
              :class="[
                'px-3 py-2 rounded-md text-sm font-medium flex items-center gap-2 transition-colors',
                route.path === item.to
                  ? 'bg-white/15 text-white'
                  : 'text-slate-300 hover:text-white hover:bg-slate-800'
              ]"
              @click="handleNavClick(item.to)"
            >
              <component :is="item.icon" class="h-4 w-4" />
              <span>{{ item.label }}</span>
            </router-link>
          </nav>

        </div>

        <!-- Right: Organization + Notifications + User menu -->
        <div class="flex items-center gap-2">
          <!-- Organization name (desktop) -->
          <div class="hidden min-[1240px]:flex items-center gap-2 text-sm text-slate-200 mr-1">
            <Building2 class="h-4 w-4" />
            <span class="max-w-40 truncate">{{ auth.organization?.name }}</span>
          </div>

          <!-- Support -->
          <Tooltip text="Поддержка" side="bottom" content-class="bg-slate-700 text-slate-100 border-slate-600">
            <Button
              variant="ghost"
              size="icon"
              class="text-slate-300 hover:text-white hover:bg-slate-800"
              data-tutorial="help-btn"
              @click="router.push('/support')"
            >
              <HelpCircle class="h-5 w-5" />
            </Button>
          </Tooltip>

          <!-- Notifications -->
          <NotificationBell />

          <!-- Mobile: avatar + BottomSheet -->
          <template v-if="isMobile">
          <Button variant="ghost" class="relative h-9 w-9 rounded-full hover:bg-slate-800" @click="isProfileOpen = true">
            <Avatar class="h-9 w-9">
              <AvatarFallback class="bg-primary/20 text-primary-foreground">
                {{ userInitial }}
              </AvatarFallback>
            </Avatar>
          </Button>
          <BottomSheet v-model="isProfileOpen" label="Профиль">
            <div class="px-4 py-3 border-b flex-shrink-0">
              <p class="text-base font-medium text-foreground">{{ auth.name }}</p>
              <p class="text-sm text-muted-foreground">{{ auth.email }}</p>
            </div>
            <nav class="px-2 py-2 space-y-1">
              <button
                class="w-full text-left px-3 py-3 rounded-md text-base flex items-center gap-3 text-foreground hover:bg-muted transition-colors"
                @click="isProfileOpen = false; router.push('/profile')"
              >
                <User class="h-5 w-5" />
                Профиль
              </button>
              <button
                v-if="canManageMembers"
                class="w-full text-left px-3 py-3 rounded-md text-base flex items-center gap-3 text-foreground hover:bg-muted transition-colors"
                @click="isProfileOpen = false; router.push('/members')"
              >
                <Users class="h-5 w-5" />
                Штат
              </button>
              <button
                v-if="canManageMembers"
                class="w-full text-left px-3 py-3 rounded-md text-base flex items-center gap-3 text-foreground hover:bg-muted transition-colors"
                @click="isProfileOpen = false; router.push('/fleet')"
              >
                <Truck class="h-5 w-5" />
                Автопарк
              </button>
              <button
                v-if="canManageMembers"
                class="w-full text-left px-3 py-3 rounded-md text-base flex items-center gap-3 text-foreground hover:bg-muted transition-colors"
                @click="isProfileOpen = false; router.push('/organization/settings')"
              >
                <Settings class="h-5 w-5" />
                Настройки
              </button>
              <Separator class="my-2" />
              <button
                class="w-full text-left px-3 py-3 rounded-md text-base flex items-center gap-3 text-destructive hover:bg-destructive/10 transition-colors"
                @click="isProfileOpen = false; logout()"
              >
                <LogOut class="h-5 w-5" />
                Выйти
              </button>
            </nav>
          </BottomSheet>
          </template>

          <!-- Desktop: DropdownMenu -->
          <DropdownMenu v-else v-model:open="isProfileOpen">
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" class="relative h-9 w-9 rounded-full hover:bg-slate-800">
                <Avatar class="h-9 w-9">
                  <AvatarFallback class="bg-primary/20 text-primary-foreground">
                    {{ userInitial }}
                  </AvatarFallback>
                </Avatar>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="w-56 bg-slate-800 border-slate-700 text-slate-100">
              <DropdownMenuLabel>
                <div class="flex flex-col space-y-1">
                  <p class="text-sm font-medium text-slate-100">{{ auth.name }}</p>
                  <p class="text-xs text-slate-400">{{ auth.email }}</p>
                </div>
              </DropdownMenuLabel>
              <DropdownMenuSeparator class="bg-slate-700" />
              <DropdownMenuItem class="text-slate-200 focus:bg-slate-700 focus:text-white" @click="router.push('/profile')">
                <User class="mr-2 h-4 w-4" />
                Профиль
              </DropdownMenuItem>
              <DropdownMenuItem
                v-if="canManageMembers"
                class="text-slate-200 focus:bg-slate-700 focus:text-white"
                @click="router.push('/members')"
              >
                <Users class="mr-2 h-4 w-4" />
                Штат
              </DropdownMenuItem>
              <DropdownMenuItem
                v-if="canManageMembers"
                class="text-slate-200 focus:bg-slate-700 focus:text-white"
                @click="router.push('/fleet')"
              >
                <Truck class="mr-2 h-4 w-4" />
                Автопарк
              </DropdownMenuItem>
              <DropdownMenuItem
                v-if="canManageMembers"
                class="text-slate-200 focus:bg-slate-700 focus:text-white"
                @click="router.push('/organization/settings')"
              >
                <Settings class="mr-2 h-4 w-4" />
                Настройки
              </DropdownMenuItem>
              <DropdownMenuSeparator class="bg-slate-700" />
              <DropdownMenuItem
                class="text-destructive focus:bg-slate-700 focus:text-destructive/80"
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
