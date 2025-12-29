import type { RouteLocationNormalized } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

type NavigationGuardReturn =
  | boolean
  | { name: string; query?: Record<string, string> }
  | undefined

export async function authGuard(
  to: RouteLocationNormalized
): Promise<NavigationGuardReturn> {
  // Skip for admin routes - they have their own auth
  if (to.meta.admin) {
    return true
  }

  const auth = useAuthStore()

  // Initialize auth state if not done yet
  if (!auth.isInitialized) {
    await auth.initialize()
  }

  // Public routes don't require auth
  if (to.meta.public) {
    // Redirect authenticated users away from login page
    if (to.name === 'login' && auth.isAuthenticated) {
      return { name: 'freight-requests' }
    }
    return true
  }

  // Not authenticated - redirect to login
  if (!auth.isAuthenticated) {
    return {
      name: 'login',
      query: { redirect: to.fullPath },
    }
  }

  return true
}

export async function orgActiveGuard(
  to: RouteLocationNormalized
): Promise<NavigationGuardReturn> {
  // Skip for admin routes
  if (to.meta.admin) {
    return true
  }

  const auth = useAuthStore()

  // Skip for public routes
  if (to.meta.public) {
    return true
  }

  // Skip for pages that don't require active org
  if (to.meta.allowInactiveOrg) {
    return true
  }

  // Check organization status
  const status = auth.organization?.status

  if (status === 'pending') {
    return { name: 'org-pending' }
  }

  if (status === 'rejected') {
    return { name: 'org-rejected' }
  }

  if (status === 'suspended') {
    return { name: 'org-suspended' }
  }

  return true
}

export async function adminGuard(
  _to: RouteLocationNormalized
): Promise<NavigationGuardReturn> {
  // Admin auth is separate - check localStorage
  const adminId = localStorage.getItem('adminId')

  if (!adminId) {
    return { name: 'admin-login' }
  }

  return true
}

export function roleGuard(
  allowedRoles: ('owner' | 'administrator' | 'employee')[]
) {
  return async (): Promise<NavigationGuardReturn> => {
    const auth = useAuthStore()

    if (!auth.role || !allowedRoles.includes(auth.role)) {
      return { name: 'forbidden' }
    }

    return true
  }
}
