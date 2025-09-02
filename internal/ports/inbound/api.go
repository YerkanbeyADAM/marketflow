package inbound

import (
	"context"
	"time"

	"marketflow/internal/app/model"
)

type APIPorts interface {
	Latest
	Highest
	Lowest
	Average
	DataMode
	SystemHealth
}

type Latest interface {
	GetLatestAggregate(ctx context.Context, symbol string) (*model.MarketData, error)
	GetLatestByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error)
}

type Highest interface {
	GetHighestAggregate(ctx context.Context, symbol string) (*model.MarketData, error)
	GetHighestByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error)
	GetHighestByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error)
}

type Lowest interface {
	GetLowestAggregate(ctx context.Context, symbol string) (*model.MarketData, error)
	GetLowestByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error)
	GetLowestByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error)
}

type Average interface {
	GetAverageAggregate(ctx context.Context, symbol string) (*model.MarketData, error)
	GetAverageByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error)
	GetAverageByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error)
}

type DataMode interface {
	SetMode(mode string)
}

type SystemHealth interface {
	HealthCheck() error
}

// GET /prices/latest/{symbol} – Получить последнюю цену для данного символа.
// GET /prices/latest/{exchange}/{symbol} – Получить актуальную цену на данный символ с конкретной биржи.
// GET /prices/highest/{symbol} – Получить самую высокую цену за период.
// GET /prices/highest/{exchange}/{symbol} —
// Получите самую высокую цену за определенный период от определенной биржи.
// GET /prices/highest/{symbol}?period={duration} –
// Получите наивысшую цену за последний {duration} (например, последние 1s, 3s, 5s, 10s, 30s, 1m, 3m, 5m).
// GET /prices/highest/{exchange}/{symbol}?period={duration} –
// Получите самую высокую цену за последний {duration} от определенного обмена.
// GET /prices/lowest/{symbol} – Получить самую низкую цену за период.
// GET /prices/lowest/{exchange}/{symbol} — Получите самую низкую цену за период от конкретной биржи.
// GET /prices/lowest/{symbol}?period={duration} – Получите самую низкую цену за последний {duration}.
// GET /prices/lowest/{exchange}/{symbol}?period={duration} –
// Получите самую низкую цену за последний {duration} от конкретной биржи.
// GET /prices/average/{symbol} – Получить среднюю цену за период.
// GET /prices/average/{exchange}/{symbol} — Получить среднюю цену за период с конкретной биржи.
// GET /prices/average/{exchange}/{symbol}?period={duration} —
// Получить среднюю цену за последний {duration} от конкретной биржи
// Data Mode API  API режима данных
// POST /mode/test – Переключиться в тестовый режим (использовать сгенерированные данные).
// POST /mode/live – Переключение в режим Live (получение данных из предоставленных программ).
// System Health  Работоспособность системы
// GET /health — возвращает состояние системы (например, подключения, доступность Redis).
