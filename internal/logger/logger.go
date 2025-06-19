package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

type Env string

const (
	EnvLocal Env = "local"
	EnvDev   Env = "dev"
	EnvProd  Env = "prod"
)

func Init(env Env) {
	var logger *slog.Logger

	switch env {
	case EnvLocal:
		logger = slog.New(
			tint.NewHandler(os.Stdout, &tint.Options{
				Level:      slog.LevelDebug,
				TimeFormat: time.Kitchen, // 3:04PM вместо длинной даты
				AddSource:  true,         // Показывать файл:строку
				NoColor:    false,        // Цвета в консоли
			}),
		)
	case EnvDev:
		// JSON для dev окружения, но с отладкой
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		}))
	case EnvProd:
		// Компактный JSON для продакшена
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	default:
		panic("unknown environment: " + string(env))
	}

	slog.SetDefault(logger)
}
