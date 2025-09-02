package outbound

import (
	"context"
	"time"

	"marketflow/internal/app/model"

	"github.com/redis/go-redis/v9"
)

type RedisPort interface {
	Redis_Method
	Redis_Conn
	Redis_Health
}

type Redis_Method interface {
	SaveMarketDataWithHistory(ctx context.Context, data *model.MarketData, duration time.Duration) error
	Get(ctx context.Context, symbol string) (*model.MarketData, error)
	GetLatestAggregate(ctx context.Context, symbol string) (*model.MarketData, error)
	GetLatestByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error)

	GetHighestAggregate(ctx context.Context, symbol string) (*model.MarketData, error)
	GetHighestByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error)
	GetHighestByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error)

	GetLowestAggregate(ctx context.Context, symbol string) (*model.MarketData, error)
	GetLowestByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error)
	GetLowestByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error)

	GetAverageAggregate(ctx context.Context, symbol string) (*model.MarketData, error)
	GetAverageByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error)
	GetAverageByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error)
}

type Redis_Conn interface {
	RedisDB() *redis.Client
	Close() error
}

type Redis_Health interface {
	Ping_Redis(ctx context.Context) error
}
