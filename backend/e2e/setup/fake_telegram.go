package setup

import "sync"

// RecordedTelegram — одно «отправленное» Telegram-сообщение fake-клиента.
type RecordedTelegram struct {
	ChatID     int64
	Text       string
	ButtonText string
	ButtonURL  string
}

// FakeTelegramSender реализует handlers.TelegramSender: записывает отправки
// вместо реального HTTP к api.telegram.org. Общий на suite — тесты ассертят
// через SentTo + substring, а не абсолютные счётчики.
type FakeTelegramSender struct {
	mu   sync.Mutex
	sent []RecordedTelegram
}

func (f *FakeTelegramSender) SendMessageWithButton(chatID int64, text, buttonText, buttonURL string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, RecordedTelegram{
		ChatID:     chatID,
		Text:       text,
		ButtonText: buttonText,
		ButtonURL:  buttonURL,
	})
	return nil
}

// Sent возвращает копию всех записанных отправок.
func (f *FakeTelegramSender) Sent() []RecordedTelegram {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]RecordedTelegram, len(f.sent))
	copy(out, f.sent)
	return out
}

// SentTo возвращает отправки на конкретный chat id.
func (f *FakeTelegramSender) SentTo(chatID int64) []RecordedTelegram {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []RecordedTelegram
	for _, m := range f.sent {
		if m.ChatID == chatID {
			out = append(out, m)
		}
	}
	return out
}
