package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"marketflow/internal/app"
	"marketflow/internal/app/model"
)

type Repo struct {
	Conn *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{Conn: db}
}

func (r *Repo) SaveAvg(ctx context.Context, avg float64) error {
	return nil
}

func (r *Repo) Ping_DB(ctx context.Context) error {
	return r.Conn.PingContext(ctx)
}

func (r *Repo) SaveAggregated(ctx context.Context, aggs []model.AggregatedData) error {
	tx, err := r.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO market_data (symbol, exchange, timestamp, average_price, min_price, max_price)
		VALUES ($1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, a := range aggs {
		cleanExchange := normalizeExchange(a.Exchange)
		cleanSymbol := normalizeSymbol(a.Symbol)
		_, err := stmt.ExecContext(ctx, cleanSymbol, cleanExchange, a.Timestamp, a.AvgPrice, a.MinPrice, a.MaxPrice)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (r *Repo) GetAverageByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error) {
	var data model.MarketData

	query := `
        SELECT 
            symbol,
            exchange,
            MAX(timestamp) AS timestamp,
            COALESCE(AVG(average_price), 0)
        FROM market_data
        WHERE exchange = $1 
          AND symbol = $2 
          AND timestamp >= NOW() - $3::interval
        GROUP BY symbol, exchange
    `
	err := r.Conn.QueryRowContext(ctx, query, exchange, symbol, durationToInterval(period)).
		Scan(&data.Symbol, &data.Exchange, &data.Timestamp, &data.Price)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.NotFound(fmt.Sprintf("no data for symbol %s and exchange %s", symbol, exchange))
		}
		return nil, err
	}
	return &data, nil
}

func (r *Repo) GetHighestByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error) {
	var data model.MarketData

	query := `
        SELECT 
            symbol,
            exchange,
            MAX(timestamp) AS timestamp,
            COALESCE(MAX(max_price), 0)
        FROM market_data
        WHERE exchange = $1 
          AND symbol = $2 
          AND timestamp >= NOW() - $3::interval
        GROUP BY symbol, exchange
    `
	err := r.Conn.QueryRowContext(ctx, query, exchange, symbol, durationToInterval(period)).
		Scan(&data.Symbol, &data.Exchange, &data.Timestamp, &data.Price)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.NotFound(fmt.Sprintf("no data for symbol %s and exchange %s", symbol, exchange))
		}
		return nil, err
	}
	return &data, nil
}

func (r *Repo) GetLowestByPeriod(ctx context.Context, exchange, symbol string, period time.Duration) (*model.MarketData, error) {
	var data model.MarketData

	query := `
        SELECT 
            symbol,
            exchange,
            MAX(timestamp) AS timestamp,
            COALESCE(MIN(min_price), 0)
        FROM market_data
        WHERE exchange = $1 
          AND symbol = $2 
          AND timestamp >= NOW() - $3::interval
        GROUP BY symbol, exchange
    `
	err := r.Conn.QueryRowContext(ctx, query, exchange, symbol, durationToInterval(period)).
		Scan(&data.Symbol, &data.Exchange, &data.Timestamp, &data.Price)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.NotFound(fmt.Sprintf("no data for symbol %s and exchange %s", symbol, exchange))
		}
		return nil, err
	}
	return &data, nil
}

func durationToInterval(d time.Duration) string {
	minutes := int64(d.Minutes())
	if minutes <= 0 {
		minutes = 1
	}
	return fmt.Sprintf("%d minute", minutes)
}

func normalizeExchange(exchange string) string {
	if i := strings.Index(exchange, ":"); i != -1 {
		return exchange[:i]
	}
	return exchange
}

func normalizeSymbol(symbol string) string {
	if i := strings.Index(symbol, ":"); i != -1 {
		return symbol[i+1:]
	}
	return symbol
}
