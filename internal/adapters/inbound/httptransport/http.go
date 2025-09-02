package httptransport

import (
	"context"
	"net/http"
	"strconv"

	ports "marketflow/internal/ports/inbound"
	"marketflow/pkg/logger"
)

type Server struct {
	addr   string
	router *http.ServeMux
	svc    ports.APIPorts
	logger *logger.CustomLogger
	server *http.Server
}

func NewHTTPServer(svc ports.APIPorts, port int, logger *logger.CustomLogger) *Server {
	router := newRouter(svc, logger)
	addr := ":" + strconv.Itoa(port)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	return &Server{
		addr:   addr,
		router: router,
		svc:    svc,
		logger: logger,
		server: httpServer,
	}
}

func newRouter(svc ports.APIPorts, logger *logger.CustomLogger) *http.ServeMux {
	router := http.NewServeMux()
	RegisterRouters(svc, router, logger)
	return router
}

func (s *Server) Serve() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
