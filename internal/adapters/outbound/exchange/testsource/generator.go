package testsource

import (
	"context"
	"math/rand"
	"time"

	"marketflow/internal/app/model"
)

type TestDataSource struct {
	exchanges []string
	symbols   []string
}

func NewTestSource() *TestDataSource {
	return &TestDataSource{
		exchanges: []string{"exchange1:40101", "exchange2:40102", "exchange3:40103"},
		symbols:   []string{"BTCUSDT", "DOGEUSDT", "TONUSDT", "SOLUSDT", "ETHUSDT"},
	}
}

var basePrices = map[string]float64{
	"BTCUSDT":  100_000,
	"DOGEUSDT": 0.30,
	"TONUSDT":  3.90,
	"SOLUSDT":  200,
	"ETHUSDT":  3_000,
}

func (t *TestDataSource) Stream(ctx context.Context, out chan<- model.MarketData) error {
	for _, exchange := range t.exchanges {
		go func(exchange string) {
			ts := time.Now()
			for _, symbol := range t.symbols {
				base := basePrices[symbol]
				volatility := base * 0.005
				price := base + rand.Float64()*volatility*2 - volatility

				out <- model.MarketData{
					Symbol:    symbol,
					Price:     price,
					Timestamp: ts,
					Exchange:  exchange,
				}
			}

			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					ts := time.Now()
					for _, symbol := range t.symbols {
						base := basePrices[symbol]
						volatility := base * 0.005
						price := base + rand.Float64()*volatility*2 - volatility

						out <- model.MarketData{
							Symbol:    symbol,
							Price:     price,
							Timestamp: ts,
							Exchange:  exchange,
						}
					}
				}
			}
		}(exchange)
	}

	<-ctx.Done()
	return nil
}
