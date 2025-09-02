package outbound

import (
	"context"
	"time"

	"marketflow/internal/app/model"
)

type DbPort interface {
	Ping_DB(ctx context.Context) error
	SaveAggregated(ctx context.Context, aggs []model.AggregatedData) error
	GetAverageByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error)
	GetHighestByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error)
	GetLowestByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error)
}
