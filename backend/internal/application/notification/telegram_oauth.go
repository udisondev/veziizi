package notification

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TelegramOAuthConfig конфигурация для Telegram OAuth
type TelegramOAuthConfig struct {
	BotToken    string
	BotUsername string
}

// TelegramAuthData данные от Telegram Login Widget
type TelegramAuthData struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	PhotoURL  string `json:"photo_url,omitempty"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}

// TelegramOAuth обрабатывает Telegram OAuth
type TelegramOAuth struct {
	config TelegramOAuthConfig
}

// NewTelegramOAuth создает новый обработчик OAuth
func NewTelegramOAuth(config TelegramOAuthConfig) *TelegramOAuth {
	return &TelegramOAuth{
		config: config,
	}
}

// GetWidgetData возвращает данные для Telegram Login Widget
func (t *TelegramOAuth) GetWidgetData() map[string]string {
	return map[string]string{
		"bot_username": t.config.BotUsername,
	}
}

// ValidateAuthData проверяет подлинность данных от Telegram
// https://core.telegram.org/widgets/login#checking-authorization
func (t *TelegramOAuth) ValidateAuthData(data TelegramAuthData) error {
	// Проверяем что auth_date не слишком старый (максимум 1 день)
	authTime := time.Unix(data.AuthDate, 0)
	if time.Since(authTime) > 24*time.Hour {
		return fmt.Errorf("auth data expired")
	}

	// Собираем data-check-string
	dataCheckString := t.buildDataCheckString(data)

	// Вычисляем секретный ключ: SHA256(bot_token)
	secretKey := sha256.Sum256([]byte(t.config.BotToken))

	// Вычисляем HMAC-SHA256
	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	// Сравниваем с полученным hash
	if !hmac.Equal([]byte(calculatedHash), []byte(data.Hash)) {
		return fmt.Errorf("invalid hash")
	}

	return nil
}

// buildDataCheckString собирает строку для проверки по алгоритму Telegram
func (t *TelegramOAuth) buildDataCheckString(data TelegramAuthData) string {
	// Собираем все поля кроме hash в формате key=value
	pairs := make([]string, 0)

	pairs = append(pairs, fmt.Sprintf("auth_date=%d", data.AuthDate))
	pairs = append(pairs, fmt.Sprintf("first_name=%s", data.FirstName))
	pairs = append(pairs, fmt.Sprintf("id=%d", data.ID))

	if data.LastName != "" {
		pairs = append(pairs, fmt.Sprintf("last_name=%s", data.LastName))
	}
	if data.PhotoURL != "" {
		pairs = append(pairs, fmt.Sprintf("photo_url=%s", data.PhotoURL))
	}
	if data.Username != "" {
		pairs = append(pairs, fmt.Sprintf("username=%s", data.Username))
	}

	// Сортируем по алфавиту
	sort.Strings(pairs)

	// Объединяем через \n
	return strings.Join(pairs, "\n")
}

// ParseAuthData парсит данные из query параметров
func ParseAuthData(params map[string]string) (TelegramAuthData, error) {
	data := TelegramAuthData{}

	// ID (обязательное)
	if idStr, ok := params["id"]; ok {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return data, fmt.Errorf("invalid id: %w", err)
		}
		data.ID = id
	} else {
		return data, fmt.Errorf("missing id")
	}

	// FirstName (обязательное)
	if firstName, ok := params["first_name"]; ok {
		data.FirstName = firstName
	} else {
		return data, fmt.Errorf("missing first_name")
	}

	// AuthDate (обязательное)
	if authDateStr, ok := params["auth_date"]; ok {
		authDate, err := strconv.ParseInt(authDateStr, 10, 64)
		if err != nil {
			return data, fmt.Errorf("invalid auth_date: %w", err)
		}
		data.AuthDate = authDate
	} else {
		return data, fmt.Errorf("missing auth_date")
	}

	// Hash (обязательное)
	if hash, ok := params["hash"]; ok {
		data.Hash = hash
	} else {
		return data, fmt.Errorf("missing hash")
	}

	// Опциональные поля
	if lastName, ok := params["last_name"]; ok {
		data.LastName = lastName
	}
	if username, ok := params["username"]; ok {
		data.Username = username
	}
	if photoURL, ok := params["photo_url"]; ok {
		data.PhotoURL = photoURL
	}

	return data, nil
}
