package livesource

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"marketflow/internal/app/model"
	"marketflow/pkg/logger"
)

type RealDataSource struct {
	Exchanges []string
	logger    *logger.CustomLogger
}

func NewRealDataSource(exchanges []string, logger *logger.CustomLogger) *RealDataSource {
	return &RealDataSource{
		Exchanges: exchanges,
		logger:    logger,
	}
}

func (r *RealDataSource) Stream(ctx context.Context, out chan<- model.MarketData) error {
	var wg sync.WaitGroup
	for _, address := range r.Exchanges {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			r.handleExchange(ctx, addr, out)
		}(address)
	}
	wg.Wait()
	return nil
}

func (r *RealDataSource) handleExchange(ctx context.Context, addr string, out chan<- model.MarketData) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Ошибка подключения к %s: %v\n", addr, err))
		return
	}
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	dataCh := make(chan model.MarketData, 100)

	var workerWg sync.WaitGroup
	r.startWorkers(ctx, dataCh, out, &workerWg)

	r.readFromScanner(ctx, addr, scanner, dataCh)

	close(dataCh)
	workerWg.Wait()
}

func (r *RealDataSource) startWorkers(ctx context.Context, dataCh <-chan model.MarketData, out chan<- model.MarketData, wg *sync.WaitGroup) {
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for data := range dataCh {
				select {
				case <-ctx.Done():
					return
				case out <- data:
				}
			}
		}()
	}
}

func (r *RealDataSource) readFromScanner(ctx context.Context, addr string, scanner *bufio.Scanner, dataCh chan<- model.MarketData) {
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			r.logger.Info(fmt.Sprintf("Остановлен парсинг данных от %s", addr))
			return
		default:
			var raw struct {
				Symbol    string  `json:"symbol"`
				Price     float64 `json:"price"`
				Timestamp int64   `json:"timestamp"`
			}

			line := scanner.Bytes()
			if err := json.Unmarshal(line, &raw); err != nil {
				r.logger.Error(fmt.Sprintf("Ошибка парсинга JSON от %s: %v. Строка: %s", addr, err, string(line)))
				continue
			}

			data := model.MarketData{
				Symbol:    raw.Symbol,
				Price:     raw.Price,
				Timestamp: time.UnixMilli(raw.Timestamp),
				Exchange:  addr,
			}

			select {
			case dataCh <- data:
			case <-ctx.Done():
				return
			}
		}
	}
}
