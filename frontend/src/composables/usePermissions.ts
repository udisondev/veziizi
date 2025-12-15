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

  // Organization status checks
  const isOrgActive = computed(() => auth.organization?.status === 'active')

  const isOrgPending = computed(() => auth.organization?.status === 'pending')

  const isOrgRejected = computed(() => auth.organization?.status === 'rejected')

  const isOrgSuspended = computed(
    () => auth.organization?.status === 'suspended'
  )

  // Carrier status
  const isCarrier = computed(() => auth.organization?.is_carrier ?? false)

  // Action permissions (combined role + org status)
  const canCreateFreightRequest = computed(() => isOrgActive.value)

  const canMakeOffer = computed(() => isOrgActive.value && isCarrier.value)

  const canBecomeCarrier = computed(
    () => isOrgActive.value && !isCarrier.value && canManageOrganization.value
  )

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

  const canSelectOffer = (customerOrgId: string): boolean => {
    return isOrgActive.value && isFreightRequestOwner(customerOrgId)
  }

  // Offer action permissions
  const canCreateOffer = (customerOrgId: string): boolean => {
    return (
      canMakeOffer.value &&
      !isFreightRequestOwner(customerOrgId) // Can't offer on own request
    )
  }

  const canWithdrawOffer = (carrierOrgId: string): boolean => {
    return isOrgActive.value && isOfferOwner(carrierOrgId)
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

  return {
    // Role-based
    canManageMembers,
    canManageOrganization,
    canManageInvitations,

    // Organization status
    isOrgActive,
    isOrgPending,
    isOrgRejected,
    isOrgSuspended,
    isCarrier,

    // Action permissions
    canCreateFreightRequest,
    canMakeOffer,
    canBecomeCarrier,

    // Resource ownership
    isFreightRequestOwner,
    isOfferOwner,
    isOrderParticipant,

    // FreightRequest actions
    canEditFreightRequest,
    canCancelFreightRequest,
    canSelectOffer,

    // Offer actions
    canCreateOffer,
    canWithdrawOffer,
    canConfirmOffer,

    // Order actions
    canViewOrder,
    canAddOrderMessage,
    canUploadOrderDocument,
  }
}
