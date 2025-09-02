package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"marketflow/internal/app"
	"marketflow/internal/app/model"
	"marketflow/internal/config"
	"marketflow/internal/ports/outbound"

	"github.com/redis/go-redis/v9"
)

type RedisRepo struct {
	client *redis.Client
}

func NewRedisRepo(cfg config.RedisConfig, ctx context.Context) (outbound.RedisPort, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisRepo{client: rdb}, nil
}

func (r *RedisRepo) RedisDB() *redis.Client {
	return r.client
}

func (r *RedisRepo) Close() error {
	return r.client.Close()
}

func (r *RedisRepo) Get(ctx context.Context, symbol string) (*model.MarketData, error) {
	key := fmt.Sprintf("token:%s", symbol)

	result := r.client.ZRevRangeWithScores(ctx, key, 0, 0).Val()
	if len(result) == 0 {
		return nil, fmt.Errorf("no data found for symbol: %s", symbol)
	}

	var data model.MarketData
	memberStr, ok := result[0].Member.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected type for member")
	}

	if err := json.Unmarshal([]byte(memberStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data for symbol %s: %w", symbol, err)
	}

	return &data, nil
}

func (r *RedisRepo) Ping_Redis(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

func (r *RedisRepo) SaveMarketDataWithHistory(ctx context.Context, data *model.MarketData, ttl time.Duration) error {
	historyKey := fmt.Sprintf("token:%s", data.Symbol)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	z := redis.Z{
		Score:  float64(data.Timestamp.UnixMilli()),
		Member: jsonData,
	}
	if err := r.client.ZAdd(ctx, historyKey, z).Err(); err != nil {
		return fmt.Errorf("failed to ZAdd: %w", err)
	}
	r.client.Expire(ctx, historyKey, ttl)

	latestKey := fmt.Sprintf("latest:%s", data.Symbol)
	if err := r.client.Set(ctx, latestKey, jsonData, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set aggregate latest: %w", err)
	}

	if data.Exchange != "" {
		exLatestKey := fmt.Sprintf("latest:%s:%s", data.Exchange, data.Symbol)
		if err := r.client.Set(ctx, exLatestKey, jsonData, ttl).Err(); err != nil {
			return fmt.Errorf("failed to set exchange latest: %w", err)
		}
	}

	highestKey := fmt.Sprintf("highest:%s", data.Symbol)
	if err := r.updateIfHigher(ctx, highestKey, data, ttl); err != nil {
		return err
	}
	if data.Exchange != "" {
		exHighestKey := fmt.Sprintf("highest:%s:%s", data.Exchange, data.Symbol)
		if err := r.updateIfHigher(ctx, exHighestKey, data, ttl); err != nil {
			return err
		}
	}

	lowestKey := fmt.Sprintf("lowest:%s", data.Symbol)
	if err := r.updateIfLower(ctx, lowestKey, data, ttl); err != nil {
		return err
	}
	if data.Exchange != "" {
		exLowestKey := fmt.Sprintf("lowest:%s:%s", data.Exchange, data.Symbol)
		if err := r.updateIfLower(ctx, exLowestKey, data, ttl); err != nil {
			return err
		}
	}

	avgKey := fmt.Sprintf("average:%s", data.Symbol)
	if err := r.updateAverage(ctx, avgKey, data, ttl); err != nil {
		return err
	}
	if data.Exchange != "" {
		exAvgKey := fmt.Sprintf("average:%s:%s", data.Exchange, data.Symbol)
		if err := r.updateAverage(ctx, exAvgKey, data, ttl); err != nil {
			return err
		}
	}

	return nil
}

func (r *RedisRepo) updateIfHigher(ctx context.Context, key string, data *model.MarketData, ttl time.Duration) error {
	existing, err := r.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if err == redis.Nil {
		return r.client.Set(ctx, key, dataToJSON(data), ttl).Err()
	}

	var existingData model.MarketData
	if err := json.Unmarshal([]byte(existing), &existingData); err != nil {
		return err
	}
	if data.Price > existingData.Price {
		return r.client.Set(ctx, key, dataToJSON(data), ttl).Err()
	}
	return nil
}

func (r *RedisRepo) updateIfLower(ctx context.Context, key string, data *model.MarketData, ttl time.Duration) error {
	existing, err := r.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if err == redis.Nil {
		return r.client.Set(ctx, key, dataToJSON(data), ttl).Err()
	}

	var existingData model.MarketData
	if err := json.Unmarshal([]byte(existing), &existingData); err != nil {
		return err
	}
	if data.Price < existingData.Price {
		return r.client.Set(ctx, key, dataToJSON(data), ttl).Err()
	}
	return nil
}

func (r *RedisRepo) updateAverage(ctx context.Context, key string, data *model.MarketData, ttl time.Duration) error {
	sumKey := key + ":sum"
	countKey := key + ":count"

	pipe := r.client.TxPipeline()
	sumCmd := pipe.IncrByFloat(ctx, sumKey, data.Price)
	countCmd := pipe.Incr(ctx, countKey)
	pipe.Expire(ctx, sumKey, ttl)
	pipe.Expire(ctx, countKey, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	sum := sumCmd.Val()
	count := countCmd.Val()
	avgPrice := sum / float64(count)

	avgData := *data
	avgData.Price = avgPrice
	return r.client.Set(ctx, key, dataToJSON(&avgData), ttl).Err()
}

func dataToJSON(data *model.MarketData) string {
	b, _ := json.Marshal(data)
	return string(b)
}

func (r *RedisRepo) GetLatestAggregate(ctx context.Context, symbol string) (*model.MarketData, error) {
	key := fmt.Sprintf("latest:%s", symbol)
	return r.getFromRedis(ctx, key)
}

func (r *RedisRepo) GetLatestByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error) {
	key := fmt.Sprintf("latest:%s:%s", exchange, symbol)
	return r.getFromRedis(ctx, key)
}

func (r *RedisRepo) getFromRedis(ctx context.Context, key string) (*model.MarketData, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, app.NotFound("no data yet")
		}
		return nil, app.Internal("failed to get data from redis", err)
	}

	var data model.MarketData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, app.Internal("failed to unmarshal redis data", err)
	}
	return &data, nil
}

func (r *RedisRepo) GetHighestAggregate(ctx context.Context, symbol string) (*model.MarketData, error) {
	key := fmt.Sprintf("highest:%s", symbol)
	return r.getFromRedis(ctx, key)
}

func (r *RedisRepo) GetHighestByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error) {
	key := fmt.Sprintf("highest:%s:%s", exchange, symbol)
	return r.getFromRedis(ctx, key)
}

func (r *RedisRepo) GetHighestByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error) {
	key := fmt.Sprintf("token:%s", symbol)

	now := time.Now().UnixMilli()
	from := now - period.Milliseconds()

	results, err := r.client.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", from),
		Max: fmt.Sprintf("%d", now),
	}).Result()
	if err != nil {
		return nil, app.Internal("redis ZRangeByScore error", err)
	}

	if len(results) == 0 {
		return nil, app.NotFound(fmt.Sprintf("no data found for symbol %s in the given period", symbol))
	}

	var (
		highest model.MarketData
		found   bool
	)

	for _, z := range results {
		memberStr, ok := z.Member.(string)
		if !ok {
			continue
		}

		var entry model.MarketData
		if err := json.Unmarshal([]byte(memberStr), &entry); err != nil {
			continue
		}

		if exchange != "" && entry.Exchange != exchange {
			continue
		}

		if !found || entry.Price > highest.Price {
			highest = entry
			found = true
		}
	}

	if !found {
		return nil, app.NotFound(fmt.Sprintf("no matching data for symbol %s and exchange %s", symbol, exchange))
	}

	return &highest, nil
}

func (r *RedisRepo) GetLowestAggregate(ctx context.Context, symbol string) (*model.MarketData, error) {
	key := fmt.Sprintf("lowest:%s", symbol)
	return r.getFromRedis(ctx, key)
}

func (r *RedisRepo) GetLowestByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error) {
	key := fmt.Sprintf("lowest:%s:%s", exchange, symbol)
	return r.getFromRedis(ctx, key)
}

func (r *RedisRepo) GetLowestByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error) {
	key := fmt.Sprintf("token:%s", symbol)

	now := time.Now().UnixMilli()
	from := now - period.Milliseconds()

	results, err := r.client.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", from),
		Max: fmt.Sprintf("%d", now),
	}).Result()
	if err != nil {
		return nil, app.Internal("redis ZRangeByScore error", err)
	}

	if len(results) == 0 {
		return nil, app.NotFound(fmt.Sprintf("no data found for symbol %s in the given period", symbol))
	}

	var (
		lowest model.MarketData
		found  bool
	)

	for _, z := range results {
		memberStr, ok := z.Member.(string)
		if !ok {
			continue
		}

		var entry model.MarketData
		if err := json.Unmarshal([]byte(memberStr), &entry); err != nil {
			continue
		}

		if exchange != "" && entry.Exchange != exchange {
			continue
		}

		if !found || entry.Price < lowest.Price {
			lowest = entry
			found = true
		}
	}

	if !found {
		return nil, app.NotFound(fmt.Sprintf("no matching data for symbol %s and exchange %s", symbol, exchange))
	}

	return &lowest, nil
}

func (r *RedisRepo) GetAverageAggregate(ctx context.Context, symbol string) (*model.MarketData, error) {
	key := fmt.Sprintf("average:%s", symbol)
	return r.getFromRedis(ctx, key)
}

func (r *RedisRepo) GetAverageByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error) {
	key := fmt.Sprintf("average:%s:%s", exchange, symbol)
	return r.getFromRedis(ctx, key)
}

func (r *RedisRepo) GetAverageByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error) {
	key := fmt.Sprintf("token:%s", symbol)

	now := time.Now().UnixMilli()
	from := now - period.Milliseconds()

	results, err := r.client.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", from),
		Max: fmt.Sprintf("%d", now),
	}).Result()
	if err != nil {
		return nil, app.Internal("redis ZRangeByScore error", err)
	}

	if len(results) == 0 {
		return nil, app.NotFound(fmt.Sprintf("no data found for symbol %s in the given period", symbol))
	}

	var (
		sum   float64
		count int64
		maxTS int64
	)

	for _, z := range results {
		memberStr, ok := z.Member.(string)
		if !ok {
			continue
		}

		var entry model.MarketData
		if err := json.Unmarshal([]byte(memberStr), &entry); err != nil {
			continue
		}

		if exchange != "" && entry.Exchange != exchange {
			continue
		}

		sum += entry.Price
		count++

		ts := entry.Timestamp.UnixMilli()
		if ts > maxTS {
			maxTS = ts
		}
	}

	if count == 0 {
		return nil, app.NotFound(fmt.Sprintf("no matching data for symbol %s and exchange %s", symbol, exchange))
	}

	avgPrice := sum / float64(count)

	ts := time.UnixMilli(maxTS).UTC()
	return &model.MarketData{
		Symbol:    symbol,
		Exchange:  exchange,
		Price:     avgPrice,
		Timestamp: ts,
	}, nil
}
