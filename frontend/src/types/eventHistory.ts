// Check if event is automatic (no actor)
export function isAutomaticEvent(eventType: string): boolean {
  const automaticEvents = [
    'organization.created',
    'freight_request.expired',
    'invitation.expired',
    'invitation.accepted',
    'order.created',
    'order.completed',
    'member.removed',
  ]
  return automaticEvents.includes(eventType)
}
