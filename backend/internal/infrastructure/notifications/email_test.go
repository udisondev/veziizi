package notifications

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
)

func TestNoopEmailProvider_Send(t *testing.T) {
	provider := NewNoopEmailProvider()

	msg := EmailMessage{
		To:        "test@example.com",
		Subject:   "Test Subject",
		BodyHTML:  "<p>Hello</p>",
		BodyText:  "Hello",
		EmailType: values.EmailTypeTransactional,
	}

	result, err := provider.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.MessageID == "" {
		t.Error("expected non-empty message ID")
	}

	// Проверяем что это валидный UUID
	if _, err := uuid.Parse(result.MessageID); err != nil {
		t.Errorf("expected valid UUID, got %q", result.MessageID)
	}
}

func TestNoopEmailProvider_SendWithTrackingID(t *testing.T) {
	provider := NewNoopEmailProvider()
	trackingID := uuid.New()

	msg := EmailMessage{
		To:         "test@example.com",
		Subject:    "Marketing Email",
		BodyHTML:   "<p>Promo</p>",
		BodyText:   "Promo",
		EmailType:  values.EmailTypeMarketing,
		TrackingID: &trackingID,
		Tags: map[string]string{
			"campaign": "summer_sale",
		},
	}

	result, err := provider.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}
}

func TestNewEmailProvider_DisabledReturnsNoop(t *testing.T) {
	provider := NewEmailProvider("resend", "api-key", "test@example.com", "Test", false)

	// Проверяем что это NoopEmailProvider
	if _, ok := provider.(*NoopEmailProvider); !ok {
		t.Error("expected NoopEmailProvider when disabled")
	}
}

func TestNewEmailProvider_EmptyAPIKeyReturnsNoop(t *testing.T) {
	provider := NewEmailProvider("resend", "", "test@example.com", "Test", true)

	// Проверяем что это NoopEmailProvider
	if _, ok := provider.(*NoopEmailProvider); !ok {
		t.Error("expected NoopEmailProvider when API key is empty")
	}
}

func TestNewEmailProvider_UnknownProviderReturnsNoop(t *testing.T) {
	provider := NewEmailProvider("unknown", "api-key", "test@example.com", "Test", true)

	// Проверяем что это NoopEmailProvider
	if _, ok := provider.(*NoopEmailProvider); !ok {
		t.Error("expected NoopEmailProvider for unknown provider")
	}
}

func TestNewEmailProvider_ResendReturnsResendProvider(t *testing.T) {
	provider := NewEmailProvider("resend", "re_test_api_key", "test@example.com", "Test", true)

	// Проверяем что это ResendProvider
	if _, ok := provider.(*ResendProvider); !ok {
		t.Error("expected ResendProvider when resend is configured")
	}
}

func TestNewResendProvider(t *testing.T) {
	provider := NewResendProvider("re_test_api_key", "test@example.com", "Test Name")

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}

	if provider.fromAddress != "test@example.com" {
		t.Errorf("expected fromAddress %q, got %q", "test@example.com", provider.fromAddress)
	}

	if provider.fromName != "Test Name" {
		t.Errorf("expected fromName %q, got %q", "Test Name", provider.fromName)
	}

	if provider.client == nil {
		t.Error("expected non-nil client")
	}
}

func TestEmailMessage_Fields(t *testing.T) {
	trackingID := uuid.New()

	msg := EmailMessage{
		To:         "recipient@example.com",
		Subject:    "Test Subject",
		BodyHTML:   "<p>HTML content</p>",
		BodyText:   "Text content",
		TrackingID: &trackingID,
		EmailType:  values.EmailTypeTransactional,
		ReplyTo:    "reply@example.com",
		Tags: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	if msg.To != "recipient@example.com" {
		t.Errorf("unexpected To: %s", msg.To)
	}

	if msg.Subject != "Test Subject" {
		t.Errorf("unexpected Subject: %s", msg.Subject)
	}

	if msg.BodyHTML != "<p>HTML content</p>" {
		t.Errorf("unexpected BodyHTML: %s", msg.BodyHTML)
	}

	if msg.BodyText != "Text content" {
		t.Errorf("unexpected BodyText: %s", msg.BodyText)
	}

	if msg.TrackingID == nil || *msg.TrackingID != trackingID {
		t.Error("unexpected TrackingID")
	}

	if msg.EmailType != values.EmailTypeTransactional {
		t.Errorf("unexpected EmailType: %s", msg.EmailType)
	}

	if msg.ReplyTo != "reply@example.com" {
		t.Errorf("unexpected ReplyTo: %s", msg.ReplyTo)
	}

	if len(msg.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(msg.Tags))
	}
}
