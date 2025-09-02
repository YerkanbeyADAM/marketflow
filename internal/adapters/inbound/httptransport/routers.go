package httptransport

import (
	"net/http"

	ports "marketflow/internal/ports/inbound"
	"marketflow/pkg/logger"
)

func RegisterRouters(svc ports.APIPorts, router *http.ServeMux, logger *logger.CustomLogger) {
	h, err := NewHandler(svc, logger)
	if err != nil {
		logger.Error("failed to create handler: " + err.Error())
	}

	logger.Info("Registering HTTP routes...")

	// Статическая страница
	router.HandleFunc("/", h.Index)

	// System Health
	router.HandleFunc("GET /health", h.HealthCheck)

	// Market Data API
	router.HandleFunc("GET /prices/latest/", h.LatestPrice)
	router.HandleFunc("GET /prices/highest/", h.HighestPrice)
	router.HandleFunc("GET /prices/lowest/", h.LowestPrice)
	router.HandleFunc("GET /prices/average/", h.AveragePrice)

	// Data Mode API
	router.HandleFunc("POST /mode/test", h.SetTestMode)
	router.HandleFunc("POST /mode/live", h.SetLiveMode)

	logger.Info("Routes registered successfully")
}

// Market Data API
// GET /prices/latest/{symbol} – Get the latest price for a given symbol.
// GET /prices/latest/{exchange}/{symbol} – Get the latest price for a given symbol from a specific exchange.
// GET /prices/highest/{symbol} – Get the highest price over a period.
// GET /prices/highest/{exchange}/{symbol} – Get the highest price over a period from a specific exchange.
// GET /prices/highest/{symbol}?period={duration} – Get the highest price within the last {duration} (e.g., the last 1s, 3s, 5s, 10s, 30s, 1m, 3m, 5m).
// GET /prices/highest/{exchange}/{symbol}?period={duration} – Get the highest price within the last {duration} from a specific exchange.
// GET /prices/lowest/{symbol} – Get the lowest price over a period.
// GET /prices/lowest/{exchange}/{symbol} – Get the lowest price over a period from a specific exchange.
// GET /prices/lowest/{symbol}?period={duration} – Get the lowest price within the last {duration}.
// GET /prices/lowest/{exchange}/{symbol}?period={duration} – Get the lowest price within the last {duration} from a specific exchange.
// GET /prices/average/{symbol} – Get the average price over a period.
// GET /prices/average/{exchange}/{symbol} – Get the average price over a period from a specific exchange.
// GET /prices/average/{exchange}/{symbol}?period={duration} – Get the average price within the last {duration} from a specific exchange
// Data Mode API
// POST /mode/test – Switch to Test Mode (use generated data).
// POST /mode/live – Switch to Live Mode (fetch data from provided programs).
// System Health
// GET /health - Returns system status (e.g., connections, Redis availability).
