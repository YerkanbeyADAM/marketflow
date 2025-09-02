package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"marketflow/internal/adapters/outbound/exchange/livesource"
	"marketflow/internal/adapters/outbound/exchange/testsource"
	"marketflow/internal/app/model"
	"marketflow/internal/ports/outbound"
	"marketflow/pkg/logger"
)

type App struct {
	mu      sync.Mutex
	db      outbound.DbPort
	redis   outbound.RedisPort
	timers  map[string]*time.Timer
	source  outbound.DataSource
	ctx     context.Context
	cancel  context.CancelFunc
	outChan chan model.MarketData
	logger  *logger.CustomLogger
	wg      sync.WaitGroup
}

func NewApp(db outbound.DbPort, redis outbound.RedisPort, source outbound.DataSource, logger *logger.CustomLogger) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		db:      db,
		redis:   redis,
		timers:  make(map[string]*time.Timer),
		source:  source,
		ctx:     ctx,
		cancel:  cancel,
		outChan: make(chan model.MarketData, 100),
		logger:  logger,
	}
}

func (a *App) Timers() map[string]*time.Timer {
	return a.timers
}

func (a *App) HealthCheck() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.db.Ping_DB(context.Background()); err != nil {
		return err
	}

	if err := a.redis.Ping_Redis(context.Background()); err != nil {
		return err
	}

	return nil
}

func (a *App) SetMode(mode string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancel != nil {
		a.cancel()
		a.wg.Wait()
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	a.cancel = cancel

	if mode == "live" {
		a.source = livesource.NewRealDataSource(
			[]string{"exchange1:40101", "exchange2:40102", "exchange3:40103"},
			a.logger,
		)
		fmt.Println("Switched to LIVE mode")
	} else {
		a.source = testsource.NewTestSource()
		fmt.Println("Switched to TEST mode")
	}

	go a.Ingest()
}

func (a *App) Ingest() {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.source.Stream(a.ctx, a.outChan); err != nil && !errors.Is(err, context.Canceled) {
			a.logger.Error(fmt.Sprintf("ingestion stream error: %v", err))
		}
	}()

	const workerCount = 5
	pgChan := make(chan model.MarketData, 200)
	aggData := make(map[string][]model.MarketData)
	aggTicker := time.NewTicker(time.Minute)

	for i := 0; i < workerCount; i++ {
		a.wg.Add(1)
		go func(id int) {
			defer a.wg.Done()
			for {
				select {
				case <-a.ctx.Done():
					return
				case data := <-a.outChan:
					if err := a.redis.SaveMarketDataWithHistory(a.ctx, &data, time.Minute); err != nil && !errors.Is(err, context.Canceled) {
						a.logger.Warn(fmt.Sprintf("Redis error: %v", err))
					}

					select {
					case pgChan <- data:
					case <-a.ctx.Done():
						return
					}
				}
			}
		}(i)
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer aggTicker.Stop()

		flush := func(ts time.Time) {
			if len(aggData) == 0 {
				return
			}
			aggregated := make([]model.AggregatedData, 0, len(aggData))
			for key, records := range aggData {
				if len(records) == 0 {
					delete(aggData, key)
					continue
				}
				var sum, min, max float64
				for i, rec := range records {
					sum += rec.Price
					if i == 0 || rec.Price < min {
						min = rec.Price
					}
					if i == 0 || rec.Price > max {
						max = rec.Price
					}
				}
				avg := sum / float64(len(records))

				parts := strings.SplitN(key, ":", 2)
				if len(parts) != 2 {
					delete(aggData, key)
					continue
				}

				aggregated = append(aggregated, model.AggregatedData{
					Symbol:    parts[1],
					Exchange:  parts[0],
					Timestamp: ts,
					AvgPrice:  avg,
					MinPrice:  min,
					MaxPrice:  max,
				})

				delete(aggData, key)
			}

			if len(aggregated) > 0 {
				a.saveAggregatedToPostgres(a.ctx, aggregated)
			}
		}

		for {
			select {
			case <-a.ctx.Done():
				drain := true
				for drain {
					select {
					case data := <-pgChan:
						key := fmt.Sprintf("%s:%s", data.Exchange, data.Symbol)
						aggData[key] = append(aggData[key], data)
					default:
						drain = false
					}
				}
				flush(time.Now().Truncate(time.Minute))
				return

			case data := <-pgChan:
				key := fmt.Sprintf("%s:%s", data.Exchange, data.Symbol)
				aggData[key] = append(aggData[key], data)

			case <-aggTicker.C:
				flush(time.Now().Truncate(time.Minute))
			}
		}
	}()
}

func (a *App) saveAggregatedToPostgres(ctx context.Context, data []model.AggregatedData) {
	copyData := make([]model.AggregatedData, len(data))
	copy(copyData, data)

	go func() {
		if err := a.db.SaveAggregated(ctx, copyData); err != nil {
			a.logger.Error(fmt.Sprintf("PostgreSQL save error: %v", err))
		}
	}()
}

func (a *App) GetLatestAggregate(ctx context.Context, symbol string) (*model.MarketData, error) {
	return a.redis.GetLatestAggregate(ctx, symbol)
}

func (a *App) GetLatestByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error) {
	return a.redis.GetLatestByExchange(ctx, exchange, symbol)
}

func (a *App) GetHighestAggregate(ctx context.Context, symbol string) (*model.MarketData, error) {
	return a.redis.GetHighestAggregate(ctx, symbol)
}

func (a *App) GetHighestByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error) {
	return a.redis.GetHighestByExchange(ctx, exchange, symbol)
}

func (a *App) GetHighestByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error) {
	if period <= time.Minute {
		data, err := a.redis.GetHighestByPeriod(ctx, exchange, symbol, period)
		if err == nil {
			return data, nil
		}
		a.logger.Warn(fmt.Sprintf("Redis недоступен, fallback на Postgres: %v", err))
	}
	return a.db.GetHighestByPeriod(ctx, normalizeExchange(exchange), symbol, period)
}

func (a *App) GetLowestAggregate(ctx context.Context, symbol string) (*model.MarketData, error) {
	return a.redis.GetLowestAggregate(ctx, symbol)
}

func (a *App) GetLowestByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error) {
	return a.redis.GetLowestByExchange(ctx, exchange, symbol)
}

func (a *App) GetLowestByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error) {
	if period <= time.Minute {
		data, err := a.redis.GetLowestByPeriod(ctx, exchange, symbol, period)
		if err == nil {
			return data, nil
		}
		a.logger.Warn(fmt.Sprintf("Redis недоступен, fallback на Postgres: %v", err))
	}
	return a.db.GetLowestByPeriod(ctx, normalizeExchange(exchange), symbol, period)
}

func (a *App) GetAverageAggregate(ctx context.Context, symbol string) (*model.MarketData, error) {
	return a.redis.GetAverageAggregate(ctx, symbol)
}

func (a *App) GetAverageByExchange(ctx context.Context, exchange, symbol string) (*model.MarketData, error) {
	return a.redis.GetAverageByExchange(ctx, exchange, symbol)
}

func (a *App) GetAverageByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error) {
	if period <= time.Minute {
		data, err := a.redis.GetAverageByPeriod(ctx, exchange, symbol, period)
		if err == nil {
			return data, nil
		}
		a.logger.Warn(fmt.Sprintf("Redis недоступен, fallback на Postgres: %v", err))
	}
	return a.db.GetAverageByPeriod(ctx, normalizeExchange(exchange), symbol, period)
}

func normalizeExchange(exchange string) string {
	if i := strings.Index(exchange, ":"); i != -1 {
		return exchange[:i]
	}
	return exchange
}
