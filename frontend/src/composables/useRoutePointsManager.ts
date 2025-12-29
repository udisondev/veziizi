/**
 * Composable for managing freight request route points
 * Handles CRUD operations and route constraints
 */

import { ref, type Ref } from 'vue'
import type { RoutePoint } from '@/types/freightRequest'

let uidCounter = 0

/**
 * Generates unique ID for route points
 */
export function generateRoutePointUid(): string {
  return `point_${Date.now()}_${++uidCounter}`
}

/**
 * Creates an empty route point with default values
 */
export function createEmptyRoutePoint(isFirst: boolean, isLast: boolean): RoutePoint {
  return {
    _uid: generateRoutePointUid(),
    is_loading: isFirst,
    is_unloading: isLast,
    address: '',
    date_from: '',
    date_to: undefined,
    time_from: undefined,
    time_to: undefined,
    contact_name: undefined,
    contact_phone: undefined,
    comment: undefined,
    coordinates: undefined,
  }
}

export interface UseRoutePointsManagerReturn {
  /** Route points array */
  routePoints: Ref<RoutePoint[]>
  /** Add a new route point */
  addRoutePoint: () => void
  /** Remove route point at index */
  removeRoutePoint: (index: number) => void
  /** Update route point at index */
  updateRoutePoint: (index: number, updates: Partial<RoutePoint>) => void
  /** Reorder route points */
  reorderRoutePoints: (newOrder: RoutePoint[]) => void
  /** Ensure route constraints (first=loading, last=unloading) */
  ensureRouteConstraints: () => void
  /** Reset to initial state with 2 points */
  resetRoutePoints: () => void
  /** Load points from existing data */
  loadRoutePoints: (points: RoutePoint[]) => void
}

/**
 * Composable for managing route points with constraints
 *
 * @example
 * ```ts
 * const {
 *   routePoints,
 *   addRoutePoint,
 *   removeRoutePoint,
 *   updateRoutePoint,
 * } = useRoutePointsManager()
 *
 * // Add a new intermediate point
 * addRoutePoint()
 *
 * // Update point address
 * updateRoutePoint(0, { address: 'Moscow' })
 * ```
 */
export function useRoutePointsManager(): UseRoutePointsManagerReturn {
  const routePoints = ref<RoutePoint[]>([
    createEmptyRoutePoint(true, false),
    createEmptyRoutePoint(false, true),
  ])

  /**
   * Ensures route constraints:
   * - First point is always loading, never unloading
   * - Last point is always unloading, never loading
   * - Intermediate points have both flags if neither is set
   */
  function ensureRouteConstraints(): void {
    const points = routePoints.value
    if (points.length === 0) return

    const firstPoint = points[0]
    const lastPoint = points[points.length - 1]

    // First point: always loading, never unloading
    if (firstPoint) {
      firstPoint.is_loading = true
      firstPoint.is_unloading = false
    }

    // Last point: always unloading, never loading
    if (lastPoint && lastPoint !== firstPoint) {
      lastPoint.is_unloading = true
      lastPoint.is_loading = false
    }

    // Intermediate points: if no flag set, set both
    for (let i = 1; i < points.length - 1; i++) {
      const point = points[i]
      if (point && !point.is_loading && !point.is_unloading) {
        point.is_loading = true
        point.is_unloading = true
      }
    }
  }

  function addRoutePoint(): void {
    // Add point at the end - it becomes new unloading,
    // old last point becomes intermediate
    const newPoint = createEmptyRoutePoint(false, false)
    routePoints.value.push(newPoint)
    ensureRouteConstraints()
  }

  function removeRoutePoint(index: number): void {
    if (routePoints.value.length > 2) {
      routePoints.value.splice(index, 1)
      ensureRouteConstraints()
    }
  }

  function updateRoutePoint(index: number, updates: Partial<RoutePoint>): void {
    if (routePoints.value[index]) {
      Object.assign(routePoints.value[index], updates)
    }
  }

  function reorderRoutePoints(newOrder: RoutePoint[]): void {
    routePoints.value = newOrder
    ensureRouteConstraints()
  }

  function resetRoutePoints(): void {
    routePoints.value = [
      createEmptyRoutePoint(true, false),
      createEmptyRoutePoint(false, true),
    ]
  }

  function loadRoutePoints(points: RoutePoint[]): void {
    // Add _uid for correct tracking in drag-and-drop
    // Convert ISO dates to YYYY-MM-DD for input type="date"
    routePoints.value = points.map((p): RoutePoint => ({
      ...p,
      _uid: generateRoutePointUid(),
      date_from: p.date_from?.split('T')[0] || '',
      date_to: p.date_to?.split('T')[0],
    }))
    ensureRouteConstraints()
  }

  return {
    routePoints,
    addRoutePoint,
    removeRoutePoint,
    updateRoutePoint,
    reorderRoutePoints,
    ensureRouteConstraints,
    resetRoutePoints,
    loadRoutePoints,
  }
}
