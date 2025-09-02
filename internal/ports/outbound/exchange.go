package outbound

import (
	"context"

	"marketflow/internal/app/model"
)

type DataSource interface {
	Stream(ctx context.Context, out chan<- model.MarketData) error
}
