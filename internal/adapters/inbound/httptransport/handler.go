package httptransport

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"marketflow/internal/app"
	"marketflow/internal/app/model"
	ports "marketflow/internal/ports/inbound"
	"marketflow/pkg"
	"marketflow/pkg/logger"
)

type Handler struct {
	svc       ports.APIPorts
	templates *template.Template
	logger    *logger.CustomLogger
}

func NewHandler(svc ports.APIPorts, logger *logger.CustomLogger) (*Handler, error) {
	tmpl := template.Must(template.ParseGlob("web/templates/*.html"))

	return &Handler{
		svc:       svc,
		templates: tmpl,
		logger:    logger,
	}, nil
}

var idToAddr = map[string]string{
	"exchange1": "exchange1:40101",
	"exchange2": "exchange2:40102",
	"exchange3": "exchange3:40103",
}

var addrToID = map[string]string{
	"exchange1:40101": "exchange1",
	"exchange2:40102": "exchange2",
	"exchange3:40103": "exchange3",
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Rendering index page")
	if err := h.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		h.logger.Error("Failed to render template: " + err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Health check requested")

	if err := h.svc.HealthCheck(); err != nil {
		h.logger.Error("Health check failed", err)
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	h.logger.Info("Health check passed")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *Handler) SetTestMode(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Switching to TEST mode")
	h.svc.SetMode("test")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) SetLiveMode(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Switching to LIVE mode")
	h.svc.SetMode("live")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) LatestPrice(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	var (
		symbol   string
		exchange string
		data     *model.MarketData
		err      error
	)

	if strReq := r.URL.Query().Get("period"); strReq != "" {
		h.logger.Info("Period parameter is not applicable for LatestPrice")
		pkg.WriteErrorJSON(w, http.StatusBadRequest, "Period parameter is not applicable for LatestPrice")
		return
	}

	switch len(parts) {
	case 3:
		symbol = strings.ToUpper(parts[2])

		if err := validateSymbol(symbol); err != nil {
			h.logger.Warn("Invalid symbol", "symbol", symbol)
			pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		data, err = h.svc.GetLatestAggregate(r.Context(), symbol)

	case 4:
		exchange = parts[2]
		exchange, err = validateExchange(exchange)
		if err != nil {
			h.logger.Warn("Invalid exchange", "exchange", parts[2])
			pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		symbol = strings.ToUpper(parts[3])
		if err := validateSymbol(symbol); err != nil {
			h.logger.Warn("Invalid symbol", "symbol", symbol)
			pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		data, err = h.svc.GetLatestByExchange(r.Context(), exchange, symbol)

	default:
		pkg.WriteErrorJSON(w, http.StatusBadRequest, "Invalid path")
		return
	}

	if err != nil {
		if appErr, ok := app.IsAppError(err); ok {
			h.logger.Warn("LatestPrice error", "symbol", symbol, "exchange", exchange, "error", appErr.Message)
			pkg.WriteErrorJSON(w, appErr.Code, appErr.Message)
			return
		}

		h.logger.Error("Unexpected error", "symbol", symbol, "exchange", exchange, "error", err)
		pkg.WriteErrorJSON(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if shortID, ok := addrToID[data.Exchange]; ok {
		data.Exchange = shortID
	}

	pkg.WriteJSON(w, http.StatusOK, data)
}

func (h *Handler) HighestPrice(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	query := r.URL.Query()

	var (
		symbol   string
		exchange string
		data     *model.MarketData
		err      error
	)

	period, err := parsePeriod(query.Get("period"))
	if err != nil {
		h.logger.Warn("Invalid period", "error", err)
		pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	switch len(parts) {
	case 3:
		symbol = strings.ToUpper(parts[2])

		if err := validateSymbol(symbol); err != nil {
			h.logger.Warn("Invalid symbol", "symbol", symbol)
			pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		if period > 0 {
			data, err = h.svc.GetHighestByPeriod(r.Context(), "", symbol, period)
		} else {
			data, err = h.svc.GetHighestAggregate(r.Context(), symbol)
		}

	case 4:
		exchange = parts[2]
		exchange, err = validateExchange(exchange)
		if err != nil {
			h.logger.Warn("Invalid exchange", "exchange", parts[2])
			pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		symbol = strings.ToUpper(parts[3])
		if err := validateSymbol(symbol); err != nil {
			h.logger.Warn("Invalid symbol", "symbol", symbol)
			pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		if period > 0 {
			data, err = h.svc.GetHighestByPeriod(r.Context(), exchange, symbol, period)
		} else {
			data, err = h.svc.GetHighestByExchange(r.Context(), exchange, symbol)
		}

	default:
		pkg.WriteErrorJSON(w, http.StatusBadRequest, "Invalid path")
		return
	}

	if err != nil {
		if appErr, ok := app.IsAppError(err); ok {
			h.logger.Warn("HighestPrice error", "symbol", symbol, "exchange", exchange, "error", appErr.Message)
			pkg.WriteErrorJSON(w, appErr.Code, appErr.Message)
			return
		}

		h.logger.Error("Unexpected error", "symbol", symbol, "exchange", exchange, "error", err)
		pkg.WriteErrorJSON(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if shortID, ok := addrToID[data.Exchange]; ok {
		data.Exchange = shortID
	}

	pkg.WriteJSON(w, http.StatusOK, data)
}

func (h *Handler) LowestPrice(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	query := r.URL.Query()

	var (
		symbol   string
		exchange string
		data     *model.MarketData
		err      error
	)

	period, err := parsePeriod(query.Get("period"))
	if err != nil {
		h.logger.Warn("Invalid period", "error", err)
		pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	switch len(parts) {
	case 3:
		symbol = strings.ToUpper(parts[2])

		if err := validateSymbol(symbol); err != nil {
			h.logger.Warn("Invalid symbol", "symbol", symbol)
			pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		if period > 0 {
			data, err = h.svc.GetLowestByPeriod(r.Context(), "", symbol, period)
		} else {
			data, err = h.svc.GetLowestAggregate(r.Context(), symbol)
		}

	case 4:
		exchange = parts[2]
		exchange, err = validateExchange(exchange)
		if err != nil {
			h.logger.Warn("Invalid exchange", "exchange", parts[2])
			pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		symbol = strings.ToUpper(parts[3])
		if err := validateSymbol(symbol); err != nil {
			h.logger.Warn("Invalid symbol", "symbol", symbol)
			pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		if period > 0 {
			data, err = h.svc.GetLowestByPeriod(r.Context(), exchange, symbol, period)
		} else {
			data, err = h.svc.GetLowestByExchange(r.Context(), exchange, symbol)
		}

	default:
		pkg.WriteErrorJSON(w, http.StatusBadRequest, "Invalid path")
		return
	}

	if err != nil {
		if appErr, ok := app.IsAppError(err); ok {
			h.logger.Warn("LowestPrice error", "symbol", symbol, "exchange", exchange, "error", appErr.Message)
			pkg.WriteErrorJSON(w, appErr.Code, appErr.Message)
			return
		}

		h.logger.Error("Unexpected error", "symbol", symbol, "exchange", exchange, "error", err)
		pkg.WriteErrorJSON(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if shortID, ok := addrToID[data.Exchange]; ok {
		data.Exchange = shortID
	}

	pkg.WriteJSON(w, http.StatusOK, data)
}

func (h *Handler) AveragePrice(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	query := r.URL.Query()

	var (
		symbol   string
		exchange string
		data     *model.MarketData
		err      error
	)

	period, err := parsePeriod(query.Get("period"))
	if err != nil {
		h.logger.Warn("Invalid period", "error", err)
		pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	switch len(parts) {
	case 3:
		symbol = strings.ToUpper(parts[2])

		if err := validateSymbol(symbol); err != nil {
			h.logger.Warn("Invalid symbol", "symbol", symbol)
			pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		if period > 0 {
			h.logger.Info("Cannot get average by period without exchange")
			pkg.WriteErrorJSON(w, http.StatusBadRequest, "Cannot get average by period without exchange")
			return
		}

		data, err = h.svc.GetAverageAggregate(r.Context(), symbol)

	case 4:
		exchange = parts[2]
		exchange, err = validateExchange(exchange)
		if err != nil {
			h.logger.Warn("Invalid exchange", "exchange", parts[2])
			pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		symbol = strings.ToUpper(parts[3])
		if err := validateSymbol(symbol); err != nil {
			h.logger.Warn("Invalid symbol", "symbol", symbol)
			pkg.WriteErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		if period > 0 {
			data, err = h.svc.GetAverageByPeriod(r.Context(), exchange, symbol, period)
		} else {
			data, err = h.svc.GetAverageByExchange(r.Context(), exchange, symbol)
		}

	default:
		pkg.WriteErrorJSON(w, http.StatusBadRequest, "Invalid path")
		return
	}

	if err != nil {
		if appErr, ok := app.IsAppError(err); ok {
			h.logger.Warn("AveragePrice error", "symbol", symbol, "exchange", exchange, "error", appErr.Message)
			pkg.WriteErrorJSON(w, appErr.Code, appErr.Message)
			return
		}

		h.logger.Error("Unexpected error", "symbol", symbol, "exchange", exchange, "error", err)
		pkg.WriteErrorJSON(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if shortID, ok := addrToID[data.Exchange]; ok {
		data.Exchange = shortID
	}

	pkg.WriteJSON(w, http.StatusOK, data)
}

func validateSymbol(symbol string) error {
	if symbol == "" {
		return app.BadRequest("symbol is required")
	}
	if len(symbol) > 10 {
		return app.BadRequest("symbol is too long")
	}
	for _, r := range symbol {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return app.BadRequest("symbol must be alphanumeric")
		}
	}
	return nil
}

func validateExchange(exchange string) (string, error) {
	if exchange == "" {
		return "", nil
	}
	if fullAddr, ok := idToAddr[strings.ToLower(exchange)]; ok {
		return fullAddr, nil
	}
	return "", app.BadRequest("unknown exchange")
}

func parsePeriod(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	period, err := time.ParseDuration(raw)
	if err != nil {
		return 0, app.BadRequest("invalid period format")
	}
	if period <= 0 {
		return 0, app.BadRequest("period must be positive")
	}
	return period, nil
}
