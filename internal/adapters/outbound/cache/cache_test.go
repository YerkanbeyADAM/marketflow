// internal/adapters/outbound/cache/cache_test.go
package cache

import (
	"context"
	"testing"
	"time"

	"marketflow/internal/app/model"
	"marketflow/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestRedisSetAndGet(t *testing.T) {
	ctx := context.Background()

	cfg := config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	repo, err := NewRedisRepo(cfg, ctx)
	assert.NoError(t, err)
	defer repo.Close()

	data := &model.MarketData{
		Symbol:    "TESTCOIN",
		Price:     123.45,
		Timestamp: time.Now(),
	}

	err = repo.SaveMarketDataWithHistory(ctx, data, time.Minute)
	assert.NoError(t, err)

	got, err := repo.Get(ctx, "TESTCOIN")
	assert.NoError(t, err)
	assert.Equal(t, data.Symbol, got.Symbol)
	assert.Equal(t, data.Price, got.Price)
}
