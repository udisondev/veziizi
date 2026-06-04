package messaging

// Канонические имена Redis-стримов (топиков). Publish-сторона маппит их через
// aggregateTopics/notificationTopics (см. publisher.go, notifications.go);
// subscribe-сторона (cmd/workers/*, e2e suite) обязана ссылаться на эти
// константы — опечатка в сыром литерале компилируется и стартует чисто, но
// consumer group молча читает пустой стрим, а проекция перестаёт обновляться.
const (
	TopicOrganizationEvents   = "organization.events"
	TopicFreightRequestEvents = "freightrequest.events"
	TopicReviewEvents         = "review.events"
	TopicSupportEvents        = "support.events"
	TopicNotificationEvents   = "notification.events"
	TopicNotificationTelegram = "notification.telegram"
	TopicNotificationEmail    = "notification.email"
)
