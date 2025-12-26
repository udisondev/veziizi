import { createRouter, createWebHistory } from 'vue-router'
import { authGuard, orgActiveGuard, roleGuard, adminGuard } from './guards'

declare module 'vue-router' {
  interface RouteMeta {
    public?: boolean
    allowInactiveOrg?: boolean
    admin?: boolean
    title?: string
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    // Public routes
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true, title: 'Вход' },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/RegisterView.vue'),
      meta: { public: true, title: 'Регистрация' },
    },
    {
      path: '/invitations/:token',
      name: 'accept-invitation',
      component: () => import('@/views/AcceptInvitationView.vue'),
      meta: { public: true, title: 'Принять приглашение' },
    },

    // Organization status pages
    {
      path: '/organization/pending',
      name: 'org-pending',
      component: () => import('@/views/OrgPendingView.vue'),
      meta: { allowInactiveOrg: true, title: 'На модерации' },
    },
    {
      path: '/organization/rejected',
      name: 'org-rejected',
      component: () => import('@/views/OrgRejectedView.vue'),
      meta: { allowInactiveOrg: true, title: 'Заявка отклонена' },
    },
    {
      path: '/organization/suspended',
      name: 'org-suspended',
      component: () => import('@/views/OrgSuspendedView.vue'),
      meta: { allowInactiveOrg: true, title: 'Организация приостановлена' },
    },

    // Protected routes
    {
      path: '/',
      name: 'freight-requests',
      component: () => import('@/views/FreightRequestsView.vue'),
      meta: { title: 'Заявки' },
    },
    {
      path: '/freight-requests/new',
      name: 'freight-request-new',
      component: () => import('@/views/FreightRequestNewView.vue'),
      meta: { title: 'Новая заявка' },
    },
    {
      path: '/freight-requests/:id',
      name: 'freight-request-detail',
      component: () => import('@/views/FreightRequestDetailView.vue'),
      meta: { title: 'Заявка' },
    },
    {
      path: '/freight-requests/:id/edit',
      name: 'freight-request-edit',
      component: () => import('@/views/FreightRequestEditView.vue'),
      meta: { title: 'Редактирование заявки' },
    },

    // Organization profile (public view)
    {
      path: '/organizations/:id',
      name: 'organization-profile',
      component: () => import('@/views/OrganizationProfileView.vue'),
      meta: { title: 'Профиль организации' },
    },

    // Member profile (public view)
    {
      path: '/members/:id',
      name: 'member-profile',
      component: () => import('@/views/MemberProfileView.vue'),
      meta: { title: 'Профиль сотрудника' },
    },

    // Orders
    {
      path: '/orders',
      name: 'orders',
      component: () => import('@/views/OrdersView.vue'),
      meta: { title: 'Заказы' },
    },
    {
      path: '/orders/:id',
      name: 'order-detail',
      component: () => import('@/views/OrderDetailView.vue'),
      meta: { title: 'Заказ' },
    },

    // Organization members (доступно всем сотрудникам)
    {
      path: '/members',
      name: 'members',
      component: () => import('@/views/MembersView.vue'),
      meta: { title: 'Штат' },
    },
    {
      path: '/organization/settings',
      name: 'org-settings',
      component: () => import('@/views/OrgSettingsView.vue'),
      meta: { title: 'Настройки организации' },
      beforeEnter: roleGuard(['owner', 'administrator']),
    },
    // My offers (any organization can make offers now)
    {
      path: '/my-offers',
      name: 'my-offers',
      component: () => import('@/views/MyOffersView.vue'),
      meta: { title: 'Предложения' },
    },

    // Profile
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/views/ProfileView.vue'),
      meta: { title: 'Профиль' },
    },

    // Notifications
    {
      path: '/notifications',
      name: 'notifications',
      component: () => import('@/views/NotificationsView.vue'),
      meta: { title: 'Уведомления' },
    },
    {
      path: '/notifications/settings',
      name: 'notification-settings',
      component: () => import('@/views/NotificationSettingsView.vue'),
      meta: { title: 'Настройки уведомлений' },
    },
    {
      path: '/subscriptions',
      name: 'freight-subscriptions',
      component: () => import('@/views/FreightSubscriptionsView.vue'),
      meta: { title: 'Подписки на заявки' },
    },

    // Admin routes
    {
      path: '/admin/login',
      name: 'admin-login',
      component: () => import('@/views/admin/AdminLoginView.vue'),
      meta: { public: true, admin: true, title: 'Вход администратора' },
    },
    {
      path: '/admin',
      name: 'admin-organizations',
      component: () => import('@/views/admin/AdminOrganizationsView.vue'),
      meta: { admin: true, title: 'Организации на модерации' },
      beforeEnter: adminGuard,
    },
    {
      path: '/admin/organizations/:id',
      name: 'admin-organization-detail',
      component: () => import('@/views/admin/AdminOrganizationDetailView.vue'),
      meta: { admin: true, title: 'Детали организации' },
      beforeEnter: adminGuard,
    },
    {
      path: '/admin/organizations',
      name: 'admin-organizations-list',
      component: () => import('@/views/admin/AdminOrganizationsView.vue'),
      meta: { admin: true, title: 'Организации на модерации' },
      beforeEnter: adminGuard,
    },
    {
      path: '/admin/reviews',
      name: 'admin-reviews',
      component: () => import('@/views/admin/AdminReviewsView.vue'),
      meta: { admin: true, title: 'Отзывы на модерации' },
      beforeEnter: adminGuard,
    },
    {
      path: '/admin/fraudsters',
      name: 'admin-fraudsters',
      component: () => import('@/views/admin/AdminFraudstersView.vue'),
      meta: { admin: true, title: 'Накрутчики' },
      beforeEnter: adminGuard,
    },

    // Error pages
    {
      path: '/forbidden',
      name: 'forbidden',
      component: () => import('@/views/ForbiddenView.vue'),
      meta: { public: true, title: 'Доступ запрещён' },
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFoundView.vue'),
      meta: { public: true, title: 'Страница не найдена' },
    },
  ],
})

// Global guards
router.beforeEach(authGuard)
router.beforeEach(orgActiveGuard)

// Update document title
router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} | Veziizi` : 'Veziizi'
})

export default router
