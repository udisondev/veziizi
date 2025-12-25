package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/infrastructure/notifications"
	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type notification struct {
	ID       uuid.UUID `db:"id"`
	MemberID uuid.UUID `db:"member_id"`
	Title    string    `db:"title"`
	Body     *string   `db:"body"`
	Link     *string   `db:"link"`
}

func main() {
	hours := flag.Int("hours", 24, "переотправить уведомления за последние N часов")
	dryRun := flag.Bool("dry-run", false, "только показать что будет отправлено, без реальной отправки")
	flag.Parse()

	// Загружаем .env файл (игнорируем ошибку если файла нет)
	_ = godotenv.Load()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("не удалось загрузить конфиг", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if cfg.Telegram.BotToken == "" {
		slog.Error("TELEGRAM_BOT_TOKEN не установлен")
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		slog.Error("не удалось подключиться к БД", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// Получаем уведомления за последние N часов
	since := time.Now().Add(-time.Duration(*hours) * time.Hour)

	query, args, err := psql.
		Select("n.id", "n.member_id", "n.title", "n.body", "n.link").
		From("inapp_notifications n").
		Join("notification_preferences p ON p.member_id = n.member_id").
		Where(squirrel.Gt{"n.created_at": since}).
		Where(squirrel.NotEq{"p.telegram_chat_id": nil}).
		OrderBy("n.created_at ASC").
		ToSql()
	if err != nil {
		slog.Error("не удалось построить запрос", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var notifs []notification
	if err := pgxscan.Select(ctx, pool, &notifs, query, args...); err != nil {
		slog.Error("не удалось получить уведомления", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("найдено уведомлений для переотправки",
		slog.Int("count", len(notifs)),
		slog.Int("hours", *hours))

	if len(notifs) == 0 {
		fmt.Println("Нет уведомлений для переотправки")
		return
	}

	telegramClient := notifications.NewTelegramClient(cfg.Telegram.BotToken)
	baseURL := cfg.App.BaseURL
	if baseURL == "" {
		baseURL = "https://veziizi.ru"
	}

	var sent, failed int
	for _, n := range notifs {
		// Получаем chat_id для member
		var chatID int64
		err := pool.QueryRow(ctx,
			"SELECT telegram_chat_id FROM notification_preferences WHERE member_id = $1",
			n.MemberID,
		).Scan(&chatID)
		if err != nil {
			slog.Warn("не удалось получить chat_id",
				slog.String("member_id", n.MemberID.String()),
				slog.String("error", err.Error()))
			failed++
			continue
		}

		body := ""
		if n.Body != nil {
			body = *n.Body
		}

		link := ""
		if n.Link != nil {
			link = baseURL + *n.Link
		}

		text := notifications.FormatNotification(n.Title, body)

		if *dryRun {
			fmt.Printf("\n--- Уведомление %s ---\n", n.ID)
			fmt.Printf("Chat ID: %d\n", chatID)
			fmt.Printf("Текст:\n%s\n", text)
			if link != "" {
				fmt.Printf("Кнопка: [Открыть в приложении] → %s\n", link)
			}
			sent++
			continue
		}

		if err := telegramClient.SendMessageWithButton(chatID, text, "Открыть в приложении", link); err != nil {
			slog.Error("не удалось отправить сообщение",
				slog.Int64("chat_id", chatID),
				slog.String("error", err.Error()))
			failed++
			continue
		}

		slog.Info("сообщение отправлено",
			slog.String("id", n.ID.String()),
			slog.Int64("chat_id", chatID))
		sent++

		// Небольшая задержка чтобы не превысить лимиты Telegram API
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("\nГотово! Отправлено: %d, ошибок: %d\n", sent, failed)
}
