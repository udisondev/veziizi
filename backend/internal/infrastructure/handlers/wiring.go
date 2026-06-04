package handlers

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// GroupHandlers-функции — единственный источник списка событийных хендлеров
// каждого воркера. Используются и cmd/workers/*/main.go, и e2e suite
// (backend/e2e/setup/suite.go): раньше списки дублировались построчно без
// compile-time связи, и новый OnXxx легко было забыть в одной из копий —
// прод и e2e тогда молча расходились.
//
// e2e гоняет полный прод-набор event-driven воркеров, включая notification-путь
// (notification-dispatcher — legacy NoPublishHandlerFunc, регистрируется через
// router.AddConsumerHandler, как в pkg/worker). Вне e2e остаются только
// scheduled-воркеры (review-activator, dedup-cleanup и т.п.) — у них нет
// событийного пути.

func MembersGroupHandlers(h *MembersHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnMemberAdded),
		cqrs.NewGroupEventHandler(h.OnMemberRemoved),
		cqrs.NewGroupEventHandler(h.OnMemberRoleChanged),
		cqrs.NewGroupEventHandler(h.OnMemberBlocked),
		cqrs.NewGroupEventHandler(h.OnMemberUnblocked),
	}
}

func OrganizationsGroupHandlers(h *OrganizationsHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnCreated),
		cqrs.NewGroupEventHandler(h.OnApproved),
		cqrs.NewGroupEventHandler(h.OnRejected),
		cqrs.NewGroupEventHandler(h.OnSuspended),
		cqrs.NewGroupEventHandler(h.OnUpdated),
	}
}

func InvitationsGroupHandlers(h *InvitationsHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnInvitationCreated),
		cqrs.NewGroupEventHandler(h.OnInvitationAccepted),
		cqrs.NewGroupEventHandler(h.OnInvitationExpired),
		cqrs.NewGroupEventHandler(h.OnInvitationCancelled),
	}
}

func PendingOrganizationsGroupHandlers(h *PendingOrganizationsHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnCreated),
		cqrs.NewGroupEventHandler(h.OnApproved),
		cqrs.NewGroupEventHandler(h.OnRejected),
	}
}

func VehiclesGroupHandlers(h *VehiclesHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnAdded),
		cqrs.NewGroupEventHandler(h.OnUpdated),
		cqrs.NewGroupEventHandler(h.OnVerified),
		cqrs.NewGroupEventHandler(h.OnRejected),
		cqrs.NewGroupEventHandler(h.OnArchived),
	}
}

func FreightRequestsGroupHandlers(h *FreightRequestsHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnCreated),
		cqrs.NewGroupEventHandler(h.OnUpdated),
		cqrs.NewGroupEventHandler(h.OnReassigned),
		cqrs.NewGroupEventHandler(h.OnCancelled),
		cqrs.NewGroupEventHandler(h.OnExpired),
		cqrs.NewGroupEventHandler(h.OnOfferMade),
		cqrs.NewGroupEventHandler(h.OnOfferWithdrawn),
		cqrs.NewGroupEventHandler(h.OnOfferSelected),
		cqrs.NewGroupEventHandler(h.OnOfferRejected),
		cqrs.NewGroupEventHandler(h.OnOfferConfirmed),
		cqrs.NewGroupEventHandler(h.OnOfferDeclined),
		cqrs.NewGroupEventHandler(h.OnOfferUnselected),
		cqrs.NewGroupEventHandler(h.OnOfferCancelledWithRequest),
		cqrs.NewGroupEventHandler(h.OnCustomerCompleted),
		cqrs.NewGroupEventHandler(h.OnCarrierCompleted),
		cqrs.NewGroupEventHandler(h.OnFreightRequestCompleted),
		cqrs.NewGroupEventHandler(h.OnReviewLeft),
		cqrs.NewGroupEventHandler(h.OnCancelledAfterConfirmed),
		cqrs.NewGroupEventHandler(h.OnCarrierMemberReassigned),
		cqrs.NewGroupEventHandler(h.OnCarrierInvited),
	}
}

func FraudsterGroupHandlers(h *FraudsterHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnFraudsterMarked),
		cqrs.NewGroupEventHandler(h.OnFraudsterUnmarked),
	}
}

func ReviewReceiverGroupHandlers(h *ReviewReceiverHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnReviewLeft),
		cqrs.NewGroupEventHandler(h.OnReviewEdited),
	}
}

func ReviewAnalyzerGroupHandlers(h *ReviewAnalyzerHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnReviewReceived),
	}
}

func ReviewsProjectionGroupHandlers(h *ReviewsProjectionHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnReceived),
		cqrs.NewGroupEventHandler(h.OnEdited),
		cqrs.NewGroupEventHandler(h.OnAnalyzed),
		cqrs.NewGroupEventHandler(h.OnApproved),
		cqrs.NewGroupEventHandler(h.OnRejected),
		cqrs.NewGroupEventHandler(h.OnActivated),
		cqrs.NewGroupEventHandler(h.OnDeactivated),
	}
}

func SupportTicketsGroupHandlers(h *SupportTicketsHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnTicketCreated),
		cqrs.NewGroupEventHandler(h.OnMessageAdded),
		cqrs.NewGroupEventHandler(h.OnTicketClosed),
		cqrs.NewGroupEventHandler(h.OnTicketReopened),
	}
}

func SupportAdminNotifierGroupHandlers(h *SupportAdminNotifierHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnTicketCreated),
		cqrs.NewGroupEventHandler(h.OnMessageAdded),
	}
}

func TelegramSenderGroupHandlers(h *TelegramSenderHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnTelegramNotification),
	}
}

func EmailSenderGroupHandlers(h *EmailSenderHandler) []cqrs.GroupEventHandler {
	return []cqrs.GroupEventHandler{
		cqrs.NewGroupEventHandler(h.OnEmailNotification),
	}
}
