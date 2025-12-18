import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

export function usePermissions() {
  const auth = useAuthStore()

  // Role-based permissions
  const canManageMembers = computed(
    () => auth.role === 'owner' || auth.role === 'administrator'
  )

  const canManageOrganization = computed(
    () => auth.role === 'owner' || auth.role === 'administrator'
  )

  const canManageInvitations = computed(
    () => auth.role === 'owner' || auth.role === 'administrator'
  )

  const canViewHistory = computed(
    () => auth.role === 'owner' || auth.role === 'administrator'
  )

  // Organization status checks
  const isOrgActive = computed(() => auth.organization?.status === 'active')

  const isOrgPending = computed(() => auth.organization?.status === 'pending')

  const isOrgRejected = computed(() => auth.organization?.status === 'rejected')

  const isOrgSuspended = computed(
    () => auth.organization?.status === 'suspended'
  )

  // Action permissions (combined role + org status)
  const canCreateFreightRequest = computed(() => isOrgActive.value)

  // Any active organization can make offers
  const canMakeOffer = computed(() => isOrgActive.value)

  // Resource ownership checks
  const isFreightRequestOwner = (customerOrgId: string): boolean => {
    return customerOrgId === auth.organizationId
  }

  const isOfferOwner = (carrierOrgId: string): boolean => {
    return carrierOrgId === auth.organizationId
  }

  const isOrderParticipant = (
    customerOrgId: string,
    carrierOrgId: string
  ): boolean => {
    return (
      customerOrgId === auth.organizationId ||
      carrierOrgId === auth.organizationId
    )
  }

  // FreightRequest action permissions
  const canEditFreightRequest = (customerOrgId: string, customerMemberId?: string): boolean => {
    if (!isOrgActive.value || !isFreightRequestOwner(customerOrgId)) {
      return false
    }
    // Владелец или администратор организации, либо создатель заявки
    const isOwnerOrAdmin = auth.role === 'owner' || auth.role === 'administrator'
    const isCreator = customerMemberId === auth.memberId
    return isOwnerOrAdmin || isCreator
  }

  const canCancelFreightRequest = (customerOrgId: string, customerMemberId?: string): boolean => {
    if (!isOrgActive.value || !isFreightRequestOwner(customerOrgId)) {
      return false
    }
    // Владелец или администратор организации, либо создатель заявки
    const isOwnerOrAdmin = auth.role === 'owner' || auth.role === 'administrator'
    const isCreator = customerMemberId === auth.memberId
    return isOwnerOrAdmin || isCreator
  }

  const canSelectOffer = (customerOrgId: string, customerMemberId?: string): boolean => {
    if (!isOrgActive.value || !isFreightRequestOwner(customerOrgId)) {
      return false
    }
    // Владелец или администратор организации, либо создатель заявки
    const isOwnerOrAdmin = auth.role === 'owner' || auth.role === 'administrator'
    const isCreator = customerMemberId === auth.memberId
    return isOwnerOrAdmin || isCreator
  }

  const canRejectOffer = (customerOrgId: string, customerMemberId?: string): boolean => {
    if (!isOrgActive.value || !isFreightRequestOwner(customerOrgId)) {
      return false
    }
    // Владелец или администратор организации, либо создатель заявки
    const isOwnerOrAdmin = auth.role === 'owner' || auth.role === 'administrator'
    const isCreator = customerMemberId === auth.memberId
    return isOwnerOrAdmin || isCreator
  }

  // Переназначить ответственного может только владелец или администратор
  const canReassignFreightRequest = (customerOrgId: string): boolean => {
    if (!isOrgActive.value || !isFreightRequestOwner(customerOrgId)) {
      return false
    }
    return auth.role === 'owner' || auth.role === 'administrator'
  }

  // Offer action permissions
  const canCreateOffer = (customerOrgId: string): boolean => {
    return (
      canMakeOffer.value &&
      !isFreightRequestOwner(customerOrgId) // Can't offer on own request
    )
  }

  const canWithdrawOffer = (carrierOrgId: string, carrierMemberId?: string): boolean => {
    if (!isOrgActive.value || !isOfferOwner(carrierOrgId)) {
      return false
    }
    const isCreator = carrierMemberId ? carrierMemberId === auth.memberId : false
    const isOwnerOrAdmin = auth.role === 'owner' || auth.role === 'administrator'
    return isCreator || isOwnerOrAdmin
  }

  const canConfirmOffer = (carrierOrgId: string): boolean => {
    return isOrgActive.value && isOfferOwner(carrierOrgId)
  }

  // Order action permissions
  const canViewOrder = (
    customerOrgId: string,
    carrierOrgId: string
  ): boolean => {
    return isOrderParticipant(customerOrgId, carrierOrgId)
  }

  const canAddOrderMessage = (
    customerOrgId: string,
    carrierOrgId: string
  ): boolean => {
    return isOrgActive.value && isOrderParticipant(customerOrgId, carrierOrgId)
  }

  const canUploadOrderDocument = (
    customerOrgId: string,
    carrierOrgId: string
  ): boolean => {
    return isOrgActive.value && isOrderParticipant(customerOrgId, carrierOrgId)
  }

  const canCompleteOrder = (
    customerOrgId: string,
    carrierOrgId: string
  ): boolean => {
    return isOrgActive.value && isOrderParticipant(customerOrgId, carrierOrgId)
  }

  const canCancelOrder = (
    customerOrgId: string,
    carrierOrgId: string
  ): boolean => {
    return isOrgActive.value && isOrderParticipant(customerOrgId, carrierOrgId)
  }

  const canLeaveOrderReview = (
    customerOrgId: string,
    carrierOrgId: string
  ): boolean => {
    return isOrgActive.value && isOrderParticipant(customerOrgId, carrierOrgId)
  }

  const canRemoveOrderDocument = (
    customerOrgId: string,
    carrierOrgId: string
  ): boolean => {
    return isOrgActive.value && isOrderParticipant(customerOrgId, carrierOrgId)
  }

  return {
    // Role-based
    canManageMembers,
    canManageOrganization,
    canManageInvitations,
    canViewHistory,

    // Organization status
    isOrgActive,
    isOrgPending,
    isOrgRejected,
    isOrgSuspended,

    // Action permissions
    canCreateFreightRequest,
    canMakeOffer,

    // Resource ownership
    isFreightRequestOwner,
    isOfferOwner,
    isOrderParticipant,

    // FreightRequest actions
    canEditFreightRequest,
    canCancelFreightRequest,
    canSelectOffer,
    canRejectOffer,
    canReassignFreightRequest,

    // Offer actions
    canCreateOffer,
    canWithdrawOffer,
    canConfirmOffer,

    // Order actions
    canViewOrder,
    canAddOrderMessage,
    canUploadOrderDocument,
    canCompleteOrder,
    canCancelOrder,
    canLeaveOrderReview,
    canRemoveOrderDocument,
  }
}
