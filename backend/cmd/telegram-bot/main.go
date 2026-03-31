package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/logging"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
)

const (
	botName    = "telegram-bot"
	pollPeriod = 1 * time.Second
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if cfg.Telegram.BotToken == "" {
		slog.Error("TELEGRAM_BOT_TOKEN is required")
		os.Exit(1)
	}

	logFile, err := logging.Setup(cfg.App.LogLevel, cfg.App.LogFile)
	if err != nil {
		slog.Error("failed to setup logger", "error", err)
		os.Exit(1)
	}
	if logFile != nil {
		defer func() {
			if err := logFile.Close(); err != nil {
				slog.Error("failed to close log file", "error", err)
			}
		}()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create factory IoC container - only needed dependencies will be lazily initialized
	// telegram-bot only uses NotificationService which doesn't require publisher
	f := factory.New(cfg)
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed to close factory", slog.String("error", err.Error()))
		}
	}()

	slog.Info("telegram bot connected to database")

	// Create bot
	bot := NewBot(cfg.Telegram.BotToken, f.NotificationService())

	slog.Info("telegram bot started", slog.String("bot", cfg.Telegram.BotUsername))

	// Start polling
	pollingDone := make(chan struct{})
	go func() {
		bot.StartPolling(ctx)
		close(pollingDone)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down telegram bot...")
	cancel()

	// Ждём завершения polling goroutine
	<-pollingDone
	slog.Info("telegram bot stopped")
}

// Bot представляет Telegram бота
type Bot struct {
	token            string
	service          NotificationService
	client           *http.Client
	baseURL          string
	offset           int64
	processedUpdates map[int64]struct{} // для дедупликации
}

// NotificationService интерфейс для работы с уведомлениями
type NotificationService interface {
	ConfirmLinkCode(ctx context.Context, code string, chatID int64, username string) error
}

// NewBot создает нового бота
func NewBot(token string, service NotificationService) *Bot {
	return &Bot{
		token:   token,
		service: service,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:          "https://api.telegram.org",
		offset:           0,
		processedUpdates: make(map[int64]struct{}),
	}
}

// StartPolling начинает опрос Telegram API
func (b *Bot) StartPolling(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			updates, err := b.getUpdates(ctx)
			if err != nil {
				slog.Error("failed to get updates", slog.String("error", err.Error()))
				time.Sleep(pollPeriod)
				continue
			}

			for _, update := range updates {
				b.offset = update.UpdateID + 1

				// Дедупликация - пропускаем уже обработанные update_id
				if _, exists := b.processedUpdates[update.UpdateID]; exists {
					continue
				}
				b.processedUpdates[update.UpdateID] = struct{}{}

				// Очищаем старые записи (храним последние 1000)
				if len(b.processedUpdates) > 1000 {
					b.cleanupProcessedUpdates()
				}

				b.handleUpdate(ctx, update)
			}

			time.Sleep(pollPeriod)
		}
	}
}

// Update представляет обновление от Telegram
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

// Message представляет сообщение
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      *Chat  `json:"chat"`
	Text      string `json:"text,omitempty"`
}

// User представляет пользователя Telegram
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Chat представляет чат
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// getUpdatesResponse ответ от getUpdates
type getUpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

func (b *Bot) getUpdates(ctx context.Context) ([]Update, error) {
	url := fmt.Sprintf("%s/bot%s/getUpdates?offset=%d&timeout=25", b.baseURL, b.token, b.offset)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", "error", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result getUpdatesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram API error")
	}

	return result.Result, nil
}

func (b *Bot) handleUpdate(ctx context.Context, update Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	if msg.Text == "" {
		return
	}

	chatID := msg.Chat.ID
	text := msg.Text

	// Обработка /start с кодом
	if strings.HasPrefix(text, "/start") {
		parts := strings.Fields(text)
		if len(parts) == 2 {
			code := parts[1]
			b.handleLinkCode(ctx, chatID, msg.From, code)
			return
		}

		// Просто /start без кода
		b.sendMessage(chatID, "👋 Привет! Я бот для уведомлений Vezii.\n\nЧтобы подключить уведомления:\n1. Зайдите в настройки уведомлений на сайте\n2. Нажмите \"Подключить Telegram\"\n3. Отправьте мне полученный код")
		return
	}

	// Проверяем, не код ли это (6 символов, буквы и цифры)
	if len(text) == 6 && isValidCode(text) {
		b.handleLinkCode(ctx, chatID, msg.From, text)
		return
	}

	// Неизвестная команда
	b.sendMessage(chatID, "Не понимаю. Отправьте код привязки из настроек уведомлений.")
}

func (b *Bot) handleLinkCode(ctx context.Context, chatID int64, user *User, code string) {
	username := ""
	if user != nil {
		username = user.Username
	}

	err := b.service.ConfirmLinkCode(ctx, strings.ToUpper(code), chatID, username)
	if err != nil {
		slog.Warn("failed to confirm link code",
			slog.String("code", code),
			slog.Int64("chat_id", chatID),
			slog.String("error", err.Error()),
		)
		b.sendMessage(chatID, "❌ Неверный или просроченный код.\n\nПолучите новый код в настройках уведомлений.")
		return
	}

	slog.Info("telegram linked successfully",
		slog.Int64("chat_id", chatID),
		slog.String("username", username),
	)

	b.sendMessage(chatID, "✅ Telegram успешно подключён!\n\nТеперь вы будете получать уведомления сюда.")
}

func (b *Bot) sendMessage(chatID int64, text string) {
	url := fmt.Sprintf("%s/bot%s/sendMessage", b.baseURL, b.token)

	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal message", slog.String("error", err.Error()))
		return
	}

	resp, err := b.client.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		slog.Error("failed to send message", slog.String("error", err.Error()))
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", "error", err)
		}
	}()
}

func isValidCode(s string) bool {
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z')) {
			return false
		}
	}
	return true
}

func (b *Bot) cleanupProcessedUpdates() {
	// Оставляем только последние 500 записей
	if len(b.processedUpdates) <= 500 {
		return
	}
	b.processedUpdates = make(map[int64]struct{})
}

