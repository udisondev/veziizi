package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// TelegramClient клиент для отправки сообщений через Telegram Bot API
type TelegramClient struct {
	botToken   string
	httpClient *http.Client
	baseURL    string
}

// NewTelegramClient создает новый Telegram клиент
func NewTelegramClient(botToken string) *TelegramClient {
	return &TelegramClient{
		botToken: botToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.telegram.org",
	}
}

// SendMessageRequest запрос на отправку сообщения
type SendMessageRequest struct {
	ChatID                int64                 `json:"chat_id"`
	Text                  string                `json:"text"`
	ParseMode             string                `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool                  `json:"disable_web_page_preview,omitempty"`
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// InlineKeyboardMarkup inline клавиатура для сообщения
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton кнопка inline клавиатуры
type InlineKeyboardButton struct {
	Text string `json:"text"`
	URL  string `json:"url,omitempty"`
}

// SendMessageResponse ответ от Telegram API
type SendMessageResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Description string `json:"description,omitempty"`
}

// SendMessage отправляет сообщение в чат
func (c *TelegramClient) SendMessage(chatID int64, text string) error {
	url := fmt.Sprintf("%s/bot%s/sendMessage", c.baseURL, c.botToken)

	reqBody := SendMessageRequest{
		ChatID:                chatID,
		Text:                  text,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", slog.String("error", err.Error()))
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram API error: %s (code: %d)", result.Description, result.ErrorCode)
	}

	return nil
}

// SendMessageWithButton отправляет сообщение с inline кнопкой
// Если URL не HTTPS — fallback на HTML-ссылку в тексте (Telegram требует HTTPS для кнопок)
func (c *TelegramClient) SendMessageWithButton(chatID int64, text, buttonText, buttonURL string) error {
	apiURL := fmt.Sprintf("%s/bot%s/sendMessage", c.baseURL, c.botToken)

	// Telegram требует HTTPS для inline кнопок
	useButton := buttonURL != "" && strings.HasPrefix(buttonURL, "https://")

	// Если не HTTPS — добавляем ссылку в текст
	if buttonURL != "" && !useButton {
		text += fmt.Sprintf("\n\n<a href=\"%s\">%s</a>", buttonURL, buttonText)
	}

	reqBody := SendMessageRequest{
		ChatID:                chatID,
		Text:                  text,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	}

	// Добавляем кнопку только для HTTPS
	if useButton {
		reqBody.ReplyMarkup = &InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{Text: buttonText, URL: buttonURL},
				},
			},
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", slog.String("error", err.Error()))
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram API error: %s (code: %d)", result.Description, result.ErrorCode)
	}

	return nil
}

// FormatNotification форматирует уведомление для отправки в Telegram (без ссылки — она в кнопке)
func FormatNotification(title, body string) string {
	return fmt.Sprintf("<b>%s</b>\n\n%s", escapeHTML(title), escapeHTML(body))
}

// escapeHTML экранирует специальные символы HTML
func escapeHTML(s string) string {
	return html.EscapeString(s)
}
